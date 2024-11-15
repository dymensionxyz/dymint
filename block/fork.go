package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
	sequencers "github.com/dymensionxyz/dymint/types/pb/rollapp/sequencers/types"
	"github.com/dymensionxyz/dymint/version"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

const (
	LoopInterval = 15 * time.Second
)

// MonitorForkUpdateLoop monitors the hub for fork updates in a loop
func (m *Manager) MonitorForkUpdateLoop(ctx context.Context) error {
	// if instruction already exists no need to check for fork update
	if types.InstructionExists(m.RootDir) {
		return nil
	}

	ticker := time.NewTicker(LoopInterval) // TODO make this configurable
	defer ticker.Stop()

	for {
		if err := m.checkForkUpdate(ctx); err != nil {
			m.logger.Error("Check for update.", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// checkForkUpdate checks if the hub has a fork update
func (m *Manager) checkForkUpdate(ctx context.Context) error {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	if m.shouldStopNode(rollapp, m.State.GetRevision()) {
		err = m.createInstruction(rollapp)
		if err != nil {
			return err
		}

		m.freezeNode(ctx, fmt.Errorf("fork update detected. local_block_height: %d rollapp_revision_start_height: %d local_revision: %d rollapp_revision: %d", m.State.Height(), rollapp.RevisionStartHeight, m.State.GetRevision(), rollapp.Revision))
	}

	return nil
}

func (m *Manager) createInstruction(rollapp *types.Rollapp) error {
	obsoleteDrs, err := m.SLClient.GetObsoleteDrs()
	if err != nil {
		return err
	}

	instruction := types.Instruction{
		Revision:            rollapp.Revision,
		RevisionStartHeight: rollapp.RevisionStartHeight,
		FaultyDRS:           obsoleteDrs,
	}

	err = types.PersistInstructionToDisk(m.RootDir, instruction)
	if err != nil {
		return err
	}

	return nil
}

// shouldStopNode determines if a rollapp node should be stopped based on revision criteria.
//
// This method checks two conditions to decide if a node should be stopped:
// 1. If the next state height is greater than or equal to the rollapp's revision start height.
// 2. If the block's app version (equivalent to revision) is less than the rollapp's revision
func (m *Manager) shouldStopNode(rollapp *types.Rollapp, revision uint64) bool {
	if m.State.NextHeight() >= rollapp.RevisionStartHeight && revision < rollapp.Revision {
		return true
	}
	return false
}

// forkNeeded returns true if the fork file exists
func (m *Manager) forkNeeded() (types.Instruction, bool) {
	if instruction, err := types.LoadInstructionFromDisk(m.RootDir); err == nil {
		return instruction, true
	}

	return types.Instruction{}, false
}

func (m *Manager) doFork(instruction types.Instruction) {
	consensusMsgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
	if err != nil {
		panic(fmt.Sprintf("prepare DRS upgrade messages: %v", err))
	}
	// Always bump the account sequences
	consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: authtypes.NewModuleAddress("sequencers").String()})

	err = m.createForkBlocks(instruction, consensusMsgs)
	if err != nil {
		panic(fmt.Sprintf("validate existing blocks: %v", err))
	}

	if err := m.submitForkBatch(instruction.RevisionStartHeight); err != nil {
		panic(fmt.Sprintf("ensure batch exists: %v", err))
	}
}

// prepareDRSUpgradeMessages prepares consensus messages for DRS upgrades.
// It performs version validation and generates the necessary upgrade messages for the sequencer.
//
// The function implements the following logic:
//   - If no faulty DRS version is provided (faultyDRS is nil), returns no messages
//   - Validates the current DRS version against the potentially faulty version
//   - Generates an upgrade message with the current valid DRS version
func (m *Manager) prepareDRSUpgradeMessages(obsoleteDRS []uint32) ([]proto.Message, error) {
	drsVersion, err := version.GetDRSVersion()
	if err != nil {
		return nil, err
	}

	for _, drs := range obsoleteDRS {
		if drs == drsVersion {
			return nil, gerrc.ErrCancelled.Wrapf("obsolete DRS version: %d", drs)
		}
	}

	return []proto.Message{
		&sequencers.MsgUpgradeDRS{
			Authority:  authtypes.NewModuleAddress("sequencers").String(),
			DrsVersion: uint64(drsVersion),
		},
	}, nil
}

// create the first two blocks of the new revision
// the first one should have a cons message(s)
// both should not have tx's
func (m *Manager) createForkBlocks(instruction types.Instruction, consensusMsgs []proto.Message) error {
	nextHeight := m.State.NextHeight()

	// Something already been created?
	// TODO: how is possible to have more than one already created? Crash failure should result in at most one.
	for h := instruction.RevisionStartHeight; h < nextHeight; h++ {
		b, err := m.Store.LoadBlock(h)
		if err != nil {
			return gerrc.ErrInternal.Wrapf("load stored blocks: %d", h)
		}

		if 0 < len(b.Data.Txs) {
			return gerrc.ErrInternal.Wrapf("fork block has tx: %d", h)
		}

		if (h == instruction.RevisionStartHeight) != (0 < len(b.Data.ConsensusMessages)) {
			return gerrc.ErrInternal.Wrapf("fork block has wrong num cons messages: %d", h)
		}
	}

	for h := nextHeight; h < instruction.RevisionStartHeight+2; h++ {
		if h == instruction.RevisionStartHeight {
			m.Executor.AddConsensusMsgs(consensusMsgs...)
		}
		zero := uint64(0)
		if _, _, err := m.ProduceApplyGossipBlock(context.Background(), ProduceBlockOptions{
			AllowEmpty:       true,
			MaxData:          &zero,
			NextProposerHash: nil,
		}); err != nil {
			return fmt.Errorf("produce apply gossip: h: %d : %w", h, err)
		}
	}

	return nil
}

// submitForkBatch verifies and, if necessary, creates a batch at the specified height.
// This function is critical for maintaining batch consistency in the blockchain while
// preventing duplicate batch submissions.
//
// The function performs the following operations:
//  1. Checks for an existing batch at the specified height via SLClient
//  2. If no batch exists, creates and submits a new one
func (m *Manager) submitForkBatch(height uint64) error {
	resp, err := m.SLClient.GetBatchAtHeight(height)
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return fmt.Errorf("getting batch at height: %v", err)
	}

	if resp != nil {
		return nil
	}

	if _, err = m.CreateAndSubmitBatch(m.Conf.BatchSubmitBytes, false); err != nil {
		return fmt.Errorf("creating and submitting batch: %v", err)
	}

	return nil
}
