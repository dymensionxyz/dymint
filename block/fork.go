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
	ForkMonitorMessage  = "rollapp fork detected. please rollback to height previous to rollapp_revision_start_height."
)

// MonitorForkUpdateLoop monitors the hub for fork updates in a loop
func (m *Manager) MonitorForkUpdateLoop(ctx context.Context) error {
	ticker := time.NewTicker(ForkMonitorInterval) // TODO make this configurable
	defer ticker.Stop()

	for {
		if err := m.checkForkUpdate(ForkMonitorMessage); err != nil {
			if errors.Is(err, ErrNonRecoverable) {
				return err
			}
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
	defer m.forkMu.Unlock()
	m.forkMu.Lock()

	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	var (
		nextHeight       = m.State.NextHeight()
		actualRevision   = m.State.GetLastRevisionNumber()
		expectedRevision = rollapp.GetRevisionForHeight(nextHeight)
	)

	if shouldStopNode(expectedRevision, nextHeight, actualRevision) {
		instruction, err := m.createInstruction(expectedRevision)
		if err != nil {
			return err
		}

		err = types.PersistInstructionToDisk(m.RootDir, instruction)
		if err != nil {
			return err
		}

		err = fmt.Errorf("%s  local_block_height: %d rollapp_revision_start_height: %d local_revision: %d rollapp_revision: %d", msg, m.State.Height(), expectedRevision.StartHeight, actualRevision, expectedRevision.GetRevisionNumber())
		return errors.Join(ErrNonRecoverable, err)
	}

	return nil
}

// createInstruction returns instruction with fork information
func (m *Manager) createInstruction(expectedRevision types.Revision) (types.Instruction, error) {
	obsoleteDrs, err := m.SLClient.GetObsoleteDrs()
	if err != nil {
		return types.Instruction{}, err
	}

	instruction := types.Instruction{
		Revision:            expectedRevision.GetRevisionNumber(),
		RevisionStartHeight: expectedRevision.StartHeight,
		FaultyDRS:           obsoleteDrs,
	}

	return instruction, nil
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
	return nextHeight >= expectedRevision.StartHeight && actualRevisionNumber < expectedRevision.GetRevisionNumber()
}

// getRevisionFromSL returns revision data for the specific height
func (m *Manager) getRevisionFromSL(height uint64) (types.Revision, error) {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return types.Revision{}, err
	}
	return rollapp.GetRevisionForHeight(height), nil
}

// getRevisions returns revision data for the rollapp
func (m *Manager) getRevisions() ([]types.Revision, error) {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return []types.Revision{}, err
	}
	return rollapp.GetRevisions(), nil
}

// doFork creates fork blocks and submits a new batch with them
func (m *Manager) doFork(instruction types.Instruction) error {
	// if fork (two) blocks are not produced and applied yet, produce them
	if m.State.Height() < instruction.RevisionStartHeight+1 {
		// add consensus msgs to upgrade DRS to running node version (msg is created in all cases and RDK will upgrade if necessary). If returns error if running version is deprecated.
		consensusMsgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
		if err != nil {
			return fmt.Errorf("prepare DRS upgrade messages: %v", err)
		}
		// add consensus msg to bump the account sequences in all fork cases
		consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: authtypes.NewModuleAddress("sequencers").String()})

		// create fork blocks
		err = m.createForkBlocks(instruction, consensusMsgs)
		if err != nil {
			return fmt.Errorf("validate fork blocks: %v", err)
		}
	}

	// submit fork batch including two fork blocks
	if err := m.submitForkBatch(instruction.RevisionStartHeight); err != nil {
		return fmt.Errorf("submit fork batch: %v", err)
	}

	return nil
}

// prepareDRSUpgradeMessages prepares consensus messages for DRS upgrades.
// It performs version validation and generates the necessary upgrade messages for the sequencer.
//
// The function implements the following logic:
//   - Validates the current DRS version against the potentially faulty version
//   - If DRS version used (binary version) is obsolete returns error
//   - If DRS version used (binary version) is different from rollapp params, generates an upgrade message with the binary DRS version
//   - Otherwise no upgrade messages are returned
func (m *Manager) prepareDRSUpgradeMessages(obsoleteDRS []uint32) ([]proto.Message, error) {
	drsVersion, err := version.GetDRSVersion()
	if err != nil {
		return nil, err
	}

	// if binary DRS is obsolete return error
	for _, drs := range obsoleteDRS {
		if drs == drsVersion {
			return nil, gerrc.ErrCancelled.Wrapf("obsolete DRS version: %d", drs)
		}
	}

	// same DRS message detected, no upgrade is necessary
	if m.State.RollappParams.DrsVersion == drsVersion {
		return []proto.Message{}, nil
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

// updateStateWithRevisions updates dymint stored state with rollapp forks revisions, to enable syncing (and validation) for rollapps with multiple revisions.
func (m *Manager) updateStateWithRevisions() error {

	// get revisions for rollapp from SL
	revisions, err := m.getRevisions()
	if err != nil {
		return err
	}

	// store revisions to dymint state
	m.State.SetRevisions(revisions)
	_, err = m.Store.SaveState(m.State, nil)
	return err

}

// doForkWhenNewRevision creates and submit to SL fork blocks according to next revision start height.
func (m *Manager) doForkWhenNewRevision() error {
	defer m.forkMu.Unlock()
	m.forkMu.Lock()

	// get revision next height
	expectedRevision, err := m.getRevisionFromSL(m.State.NextHeight())
	if err != nil {
		return err
	}

	// create fork batch in case it has not been submitted yet
	if m.LastSettlementHeight.Load() < expectedRevision.StartHeight {
		instruction, err := m.createInstruction(expectedRevision)
		if err != nil {
			return err
		}
		// create and submit fork batch
		err = m.doFork(instruction)
		if err != nil {
			return err
		}
	}

	// this cannot happen. it means the revision number obtained is not the same or the next revision. unable to fork.
	if expectedRevision.GetRevisionNumber() != m.State.GetLastRevisionNumber() {
		return fmt.Errorf("inconsistent expected revision number from Hub (%d != %d). Unable to fork", expectedRevision.GetRevisionNumber(), m.State.GetLastRevisionNumber())
	}

	// remove instruction file after fork
	return types.DeleteInstructionFromDisk(m.RootDir)
}
