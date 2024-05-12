package block

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/dymensionxyz/dymint/types"
)

// applyBlock applies the block to the store and the abci app.
// Contract: block and commit must be validated before calling this function!
// steps: save block -> execute block with app -> update state -> commit block to app -> update state's height and commit result.
// As the entire process can't be atomic we need to make sure the following condition apply before
// - block height is the expected block height on the store (height + 1).
// - block height is the expected block height on the app (last block height + 1).
func (m *Manager) applyBlock(block *types.Block, commit *types.Commit, blockMetaData blockMetaData) error {
	// TODO: add switch case to have defined behavior for each case.
	// validate block height
	if block.Header.Height != m.State.NextHeight() {
		return types.ErrInvalidBlockHeight
	}

	m.logger.Debug("Applying block", "height", block.Header.Height, "source", blockMetaData.source)

	// Check if the app's last block height is the same as the currently produced block height
	isBlockAlreadyApplied, err := m.isHeightAlreadyApplied(block.Header.Height)
	if err != nil {
		return fmt.Errorf("check if block is already applied: %w", err)
	}
	// In case the following true, it means we crashed after the commit and before updating the store height.
	// In that case we'll want to align the store with the app state and continue to the next block.
	if isBlockAlreadyApplied {
		err := m.UpdateStateFromApp()
		if err != nil {
			return fmt.Errorf("update state from app: %w", err)
		}
		m.logger.Debug("Aligned with app state required. Skipping to next block", "height", block.Header.Height)
		return nil
	}
	// Start applying the block assuming no inconsistency was found.
	_, err = m.Store.SaveBlock(block, commit, nil)
	if err != nil {
		return fmt.Errorf("save block: %w", err)
	}

	responses, err := m.Executor.ExecuteBlock(m.State, block)
	if err != nil {
		return fmt.Errorf("execute block: %w", err)
	}

	// Updates the state with validator changes and consensus params changes from the app
	err = m.Executor.UpdateStateFromResponses(&m.State, responses, block)
	if err != nil {
		return fmt.Errorf("update state from responses: %w", err)
	}

	dbBatch := m.Store.NewBatch()
	dbBatch, err = m.Store.SaveBlockResponses(block.Header.Height, responses, dbBatch)
	if err != nil {
		dbBatch.Discard()
		return fmt.Errorf("save block responses: %w", err)
	}

	dbBatch, err = m.Store.SaveValidators(block.Header.Height, m.State.Validators, dbBatch)
	if err != nil {
		dbBatch.Discard()
		return fmt.Errorf("save validators: %w", err)
	}

	err = dbBatch.Commit()
	if err != nil {
		return fmt.Errorf("commit batch to disk: %w", err)
	}

	// Commit block to app
	appHash, retainHeight, err := m.Executor.Commit(m.State, block, responses)
	if err != nil {
		return fmt.Errorf("commit block: %w", err)
	}

	// Update the state with the new app hash, last validators and store height from the commit.
	// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
	m.Executor.UpdateStateFromCommitResponse(&m.State, responses, appHash, block.Header.Height)
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return fmt.Errorf("final update state: %w", err)
	}

	// Prune old heights, if requested by ABCI app.
	if retainHeight > 0 {
		pruned, err := m.pruneBlocks(uint64(retainHeight))
		if err != nil {
			m.logger.Error("prune blocks", "retain_height", retainHeight, "err", err)
		} else {
			m.logger.Debug("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
		}
		m.State.BaseHeight = m.State.BaseHeight
		_, err = m.Store.SaveState(m.State, nil)
		if err != nil {
			return fmt.Errorf("final update state: %w", err)
		}
	}
	return nil
}

// isHeightAlreadyApplied checks if the block height is already applied to the app.
func (m *Manager) isHeightAlreadyApplied(blockHeight uint64) (bool, error) {
	proxyAppInfo, err := m.Executor.GetAppInfo()
	if err != nil {
		return false, errorsmod.Wrap(err, "get app info")
	}

	isBlockAlreadyApplied := uint64(proxyAppInfo.LastBlockHeight) == blockHeight

	// TODO: add switch case to validate better the current app state

	return isBlockAlreadyApplied, nil
}

func (m *Manager) validateBlock(block *types.Block, commit *types.Commit) error {
	// Currently we're assuming proposer is never nil as it's a pre-condition for
	// dymint to start
	proposer := m.SLClient.GetProposer()

	return types.ValidateProposedTransition(m.State, block, commit, proposer)
}
