package block

import (
	"context"
	"errors"
	"fmt"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	sequencers "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/version"
)

const (
	ForkMonitorInterval = 15 * time.Second
	ForkMessage         = "rollapp fork detected. please rollback to height previous to rollapp_revision_start_height."
)

// MonitorForkUpdateLoop monitors the hub for fork updates in a loop
func (m *Manager) MonitorForkUpdateLoop(ctx context.Context) error {
	ticker := time.NewTicker(ForkMonitorInterval) // TODO make this configurable
	defer ticker.Stop()

	for {
		if err := m.checkForkUpdate(ForkMessage); err != nil {
			m.logger.Error("Check for update.", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// checkForkUpdate checks if the hub has a fork update
func (m *Manager) checkForkUpdate(msg string) error {
	// if instruction exists no need to check for fork update
	if types.InstructionExists(m.RootDir) {
		return nil
	}

	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	var (
		nextHeight       = m.State.NextHeight()
		actualRevision   = m.State.GetRevision()
		expectedRevision = rollapp.GetRevisionForHeight(nextHeight)
	)
	if shouldStopNode(expectedRevision, nextHeight, actualRevision) {
		err = m.createInstruction(expectedRevision)
		if err != nil {
			return err
		}

		m.freezeNode(fmt.Errorf("%s  local_block_height: %d rollapp_revision_start_height: %d local_revision: %d rollapp_revision: %d", msg, m.State.Height(), expectedRevision.StartHeight, actualRevision, expectedRevision.Number))
	}

	return nil
}

// createInstruction writes file to disk with fork information
func (m *Manager) createInstruction(expectedRevision types.Revision) error {
	obsoleteDrs, err := m.SLClient.GetObsoleteDrs()
	if err != nil {
		return err
	}

	instruction := types.Instruction{
		Revision:            expectedRevision.Number,
		RevisionStartHeight: expectedRevision.StartHeight,
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
func shouldStopNode(
	expectedRevision types.Revision,
	nextHeight uint64,
	actualRevisionNumber uint64,
) bool {
	return nextHeight >= expectedRevision.StartHeight && actualRevisionNumber < expectedRevision.Number
}

// forkNeeded returns true if the fork file exists
func (m *Manager) forkNeeded() (types.Instruction, bool) {
	if instruction, err := types.LoadInstructionFromDisk(m.RootDir); err == nil {
		return instruction, true
	}

	return types.Instruction{}, false
}

// doFork creates fork blocks and submits a new batch with them
func (m *Manager) doFork(instruction types.Instruction) error {
	// if fork (two) blocks are not produced and applied yet, produce them
	if m.State.Height() < instruction.RevisionStartHeight+1 {
		// add consensus msgs for upgrade DRS only if current DRS is obsolete
		consensusMsgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
		if err != nil {
			panic(fmt.Sprintf("prepare DRS upgrade messages: %v", err))
		}
		// add consensus msg to bump the account sequences in all fork cases
		consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: authtypes.NewModuleAddress("sequencers").String()})

		// create fork blocks
		err = m.createForkBlocks(instruction, consensusMsgs)
		if err != nil {
			panic(fmt.Sprintf("validate existing blocks: %v", err))
		}
	}

	// submit fork batch including two fork blocks
	if err := m.submitForkBatch(instruction.RevisionStartHeight); err != nil {
		panic(fmt.Sprintf("ensure batch exists: %v", err))
	}

	return nil
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

	//	Revise already created fork blocks
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

	// create two empty blocks including consensus msgs in the first one
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

// updateStateWhenFork updates dymint state in case fork is detected
func (m *Manager) updateStateWhenFork() error {
	// in case fork is detected dymint state needs to be updated
	if instruction, forkNeeded := m.forkNeeded(); forkNeeded {
		// Set proposer to nil to force updating it from SL
		m.State.SetProposer(nil)
		// Upgrade revision on state
		m.State.RevisionStartHeight = instruction.RevisionStartHeight
		// this is necessary to pass ValidateConfigWithRollappParams when DRS upgrade is required
		if instruction.RevisionStartHeight == m.State.NextHeight() {
			m.State.SetRevision(instruction.Revision)
			drsVersion, err := version.GetDRSVersion()
			if err != nil {
				return err
			}
			m.State.RollappParams.DrsVersion = drsVersion
		}
		_, err := m.Store.SaveState(m.State, nil)
		return err
	}
	return nil
}

// forkFromInstruction checks if fork is needed, reading instruction file, and performs fork actions
func (m *Manager) forkFromInstruction() error {
	// if instruction file exists proposer needs to do fork actions (if settlement height is higher than revision height it is considered fork already happened and no need to do anything)
	instruction, forkNeeded := m.forkNeeded()
	if !forkNeeded {
		return nil
	}
	if m.RunMode == RunModeProposer {
		// it is checked again whether the node is the active proposer, since this could have changed after syncing.
		amIProposerOnSL, err := m.AmIProposerOnSL()
		if err != nil {
			return fmt.Errorf("am i proposer on SL: %w", err)
		}
		if !amIProposerOnSL {
			return fmt.Errorf("the node is no longer the proposer. please restart.")
		}
		// update revision with revision after fork
		m.State.SetRevision(instruction.Revision)
		// update sequencer in case it changed after syncing
		err = m.UpdateProposerFromSL()
		if err != nil {
			return err
		}
		// create fork batch in case it has not been submitted yet
		if m.LastSettlementHeight.Load() < instruction.RevisionStartHeight {
			err := m.doFork(instruction)
			if err != nil {
				return err
			}
		}
	}
	// remove instruction file after fork to avoid enter loop again
	err := types.DeleteInstructionFromDisk(m.RootDir)
	if err != nil {
		return fmt.Errorf("deleting instruction file: %w", err)
	}
	return nil
}
