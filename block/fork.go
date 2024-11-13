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
	if !types.InstructionExists(m.RootDir) {
		err := m.checkForkUpdate(ctx)
		if err != nil {
			return err
		}
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

		m.freezeNode(ctx, fmt.Errorf("fork update detected. local_block_height: %d rollapp_revision_start_height: %d local_revision: %d rollapp_revision: %d", m.State.Height(), rollapp.RevisionStartHeight, lastBlock.GetRevision(), rollapp.Revision))
	}

	return nil
}

func (m *Manager) createInstruction(rollapp *types.Rollapp) error {
	obsoleteDrs, err := m.SLClient.GetObsoleteDrs()
	if err != nil {
		return err
	}
	currentDRS, err := version.CurrentDRSVersion()
	if err != nil {
		return err
	}
	instruction := types.Instruction{
		Revision:            rollapp.Revision,
		RevisionStartHeight: rollapp.RevisionStartHeight,
		FaultyDRS:           obsoleteDrs,
		DRSPreFork:          currentDRS,
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
	revision := block.GetRevision()
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

// handleSequencerForkTransition handles the sequencer fork transition
func (m *Manager) handleSequencerForkTransition(instruction types.Instruction) {

	var consensusMsgs []proto.Message
	if isDRSFaulty(instruction.DRSPreFork, instruction.FaultyDRS) {
		msgs, err := m.prepareDRSUpgradeMessages(instruction.FaultyDRS)
		if err != nil {
			panic(fmt.Sprintf("prepare DRS upgrade messages: %v", err))
		}
		consensusMsgs = append(consensusMsgs, msgs...)
	}
	// Always bump the account sequences
	consensusMsgs = append(consensusMsgs, &sequencers.MsgBumpAccountSequences{Authority: authtypes.NewModuleAddress("sequencers").String()})

	err := m.handleCreationOfForkBlocks(instruction, consensusMsgs)
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

	currentDRS, err := version.CurrentDRSVersion()
	if err != nil {
		return nil, err
	}

	for _, drs := range faultyDRS {
		if drs == currentDRS {
			return nil, fmt.Errorf("running faulty DRS version %d", drs)
		}
	}

	return []proto.Message{
		&sequencers.MsgUpgradeDRS{
			Authority:  authtypes.NewModuleAddress("sequencers").String(),
			DrsVersion: uint64(currentDRS),
		},
	}, nil
}

// Block Creation Rules:
//   - Two blocks are considered in this process:
//   - First block: Contains consensus messages for the fork
//   - Second block: Should be empty (no messages or transactions)
//   - Total height increase should be 2 blocks from RevisionStartHeight
func (m *Manager) handleCreationOfForkBlocks(instruction types.Instruction, consensusMsgs []proto.Message) error {
	nextHeight := m.State.NextHeight()
	heightDiff := nextHeight - instruction.RevisionStartHeight

	// Case 1: Both blocks already exist (heightDiff == 2)
	if heightDiff == 2 {
		return m.validateExistingBlocks(instruction, 2)
	}

	// Case 2: First block exists (heightDiff == 1)
	if heightDiff == 1 {
		if err := m.validateExistingBlocks(instruction, 1); err != nil {
			return err
		}
		return m.createNewBlocks(consensusMsgs, 1) // Create only the second block
	}

	// Case 3: No blocks exist yet (heightDiff == 0)
	if heightDiff == 0 {
		return m.createNewBlocks(consensusMsgs, 2) // Create both blocks
	}

	return fmt.Errorf("unexpected height difference: %d", heightDiff)
}

// validateExistingBlocks validates one or two consecutive blocks based on
// the specified number of blocks to validate.
//
// Validation process:
//  1. For the first block (if numBlocksToValidate > 0):
//     - Verifies it contains consensus messages
//  2. For the second block (if numBlocksToValidate > 1):
//     - Verifies it does NOT contain consensus messages
//     - Verifies it does NOT contain transactions
//     - Basically, that is empty.
func (m *Manager) validateExistingBlocks(instruction types.Instruction, numBlocksToValidate uint64) error {
	if numBlocksToValidate > 0 {
		block, err := m.Store.LoadBlock(instruction.RevisionStartHeight)
		if err != nil {
			return fmt.Errorf("loading block: %v", err)
		}

		if len(block.Data.ConsensusMessages) <= 0 {
			return fmt.Errorf("expected consensus messages in block")
		}
	}

	if numBlocksToValidate > 1 {
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
	}

	return nil
}

// createNewBlocks creates new blocks with the provided consensus messages.
// If blocksToCreate is 1, it creates only the second (empty) block.
// If blocksToCreate is 2, it creates both the first block (with consensus messages)
// and the second empty block.
func (m *Manager) createNewBlocks(consensusMsgs []proto.Message, blocksToCreate uint64) error {
	// Add consensus messages regardless of blocks to create
	m.Executor.AddConsensusMsgs(consensusMsgs...)

	// Create first block with consensus messages if blocksToCreate == 2
	if blocksToCreate == 2 {
		if _, _, err := m.ProduceApplyGossipBlock(context.Background(), true); err != nil {
			return fmt.Errorf("producing first block: %v", err)
		}
	}

	// Create second empty block for both cases (blocksToCreate == 1 or 2)
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

func isDRSFaulty(drs uint32, faultyDRS []uint32) bool {
	for _, faulty := range faultyDRS {
		if drs == faulty {
			return true
		}
	}
	return false
}
