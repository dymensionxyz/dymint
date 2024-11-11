package block

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

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
	err := m.checkForkUpdate(ctx)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(LoopInterval) // TODO make this configurable
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := m.checkForkUpdate(ctx); err != nil {
				continue
			}
			if m.isFrozen() {
				return nil
			}
		}
	}
}

// checkForkUpdate checks if the hub has a fork update
func (m *Manager) checkForkUpdate(ctx context.Context) error {
	rollapp, err := m.SLClient.GetRollapp()
	if err != nil {
		return err
	}

	if m.State.Height() == 0 {
		return nil
	}
	lastBlock, err := m.Store.LoadBlock(m.State.Height())
	if err != nil {
		return err
	}

	if m.shouldStopNode(rollapp, lastBlock) {
		err = m.createInstruction(rollapp)
		if err != nil {
			return err
		}
		err := fmt.Errorf("fork update")
		m.freezeNode(ctx, err)
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
func (m *Manager) shouldStopNode(rollapp *types.Rollapp, block *types.Block) bool {

	revision := block.Header.Version.App
	if m.State.NextHeight() >= rollapp.RevisionStartHeight && revision < rollapp.Revision {
		m.logger.Info(
			"Freezing node due to fork update",
			"local_block_height",
			m.State.Height(),
			"rollapp_revision_start_height",
			rollapp.RevisionStartHeight,
			"local_revision",
			revision,
			"rollapp_revision",
			rollapp.Revision,
		)
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

// handleSequencerForkTransition handles the sequencer fork transition
func (m *Manager) handleSequencerForkTransition(instruction types.Instruction) {
	consensusMsgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
	if err != nil {
		panic(fmt.Sprintf("prepare DRS upgrade messages: %v", err))
	}

	// Always bump the account sequences
	consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: "gov"})

	err = m.handleForkBlockCreation(instruction, consensusMsgs)
	if err != nil {
		panic(fmt.Sprintf("validate existing blocks: %v", err))
	}

	if err := m.handleForkBatchSubmission(instruction.RevisionStartHeight); err != nil {
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
func (m *Manager) prepareDRSUpgradeMessages(faultyDRS []uint32) ([]proto.Message, error) {
	if faultyDRS == nil {
		return nil, nil
	}

	actualDRS, err := strconv.ParseUint(version.DrsVersion, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("converting DRS version to int: %v", err)
	}

	for _, drs := range faultyDRS {
		if drs == uint32(actualDRS) {
			return nil, fmt.Errorf("running faulty DRS version %d", drs)
		}
	}

	return []proto.Message{
		&sequencers.MsgUpgradeDRS{
			DrsVersion: uint64(actualDRS),
		},
	}, nil
}

// handleForkBlockCreation manages the block creation process during a fork transition.
//
// The function implements the following logic:
//  1. Checks if blocks for the fork transition have already been created by comparing heights
//  2. If blocks exist (NextHeight == RevisionStartHeight + 2), validates their state
//  3. If blocks don't exist, triggers the creation of new blocks with the provided consensus messages
//
// Block Creation Rules:
//   - Two blocks are considered in this process:
//   - First block: Contains consensus messages for the fork
//   - Second block: Should be empty (no messages or transactions)
//   - Total height increase should be 2 blocks from RevisionStartHeight
func (m *Manager) handleForkBlockCreation(instruction types.Instruction, consensusMsgs []proto.Message) error {
	if m.State.NextHeight() == instruction.RevisionStartHeight+2 {
		return m.validateExistingBlocks(instruction)
	}

	return m.createNewBlocks(consensusMsgs)
}

// validateExistingBlocks performs validation checks on a pair of consecutive blocks
// during the sequencer fork transition process.
//
// The function performs the following validations:
//  1. Verifies that the initial block at RevisionStartHeight exists and contains consensus messages
//  2. Confirms that the subsequent block exists and is empty (no consensus messages or transactions)
func (m *Manager) validateExistingBlocks(instruction types.Instruction) error {
	block, err := m.Store.LoadBlock(instruction.RevisionStartHeight)
	if err != nil {
		return fmt.Errorf("loading block: %v", err)
	}

	if len(block.Data.ConsensusMessages) <= 0 {
		return fmt.Errorf("expected consensus messages in block")
	}

	nextBlock, err := m.Store.LoadBlock(instruction.RevisionStartHeight + 1)
	if err != nil {
		return fmt.Errorf("loading next block: %v", err)
	}

	if len(nextBlock.Data.ConsensusMessages) > 0 {
		return fmt.Errorf("unexpected consensus messages in next block")
	}

	if len(nextBlock.Data.Txs) > 0 {
		return fmt.Errorf("unexpected transactions in next block")
	}

	return nil
}

// createNewBlocks creates new blocks with the provided consensus messages
func (m *Manager) createNewBlocks(consensusMsgs []proto.Message) error {
	m.Executor.AddConsensusMsgs(consensusMsgs...)

	// Create first block with consensus messages
	if _, _, err := m.ProduceApplyGossipBlock(context.Background(), true); err != nil {
		return fmt.Errorf("producing first block: %v", err)
	}

	// Create second empty block
	if _, _, err := m.ProduceApplyGossipBlock(context.Background(), true); err != nil {
		return fmt.Errorf("producing second block: %v", err)
	}

	return nil
}

// handleForkBatchSubmission verifies and, if necessary, creates a batch at the specified height.
// This function is critical for maintaining batch consistency in the blockchain while
// preventing duplicate batch submissions.
//
// The function performs the following operations:
//  1. Checks for an existing batch at the specified height via SLClient
//  2. If no batch exists, creates and submits a new one
func (m *Manager) handleForkBatchSubmission(height uint64) error {
	resp, err := m.SLClient.GetBatchAtHeight(height)
	if err != nil && !errors.Is(err, gerrc.ErrNotFound) {
		return fmt.Errorf("getting batch at height: %v", err)
	}

	if resp == nil {
		if _, err := m.CreateAndSubmitBatch(m.Conf.BatchSubmitBytes, false); err != nil {
			return fmt.Errorf("creating and submitting batch: %v", err)
		}
	}

	return nil
}
