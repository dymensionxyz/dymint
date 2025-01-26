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
func (m *Manager) applyBlock(block *types.Block, commit *types.Commit, blockMetaData types.BlockMetaData) error {
	// TODO: add switch case to have defined behavior for each case.
	// validate block height
	if block.Header.Height != m.State.NextHeight() {
		return types.ErrInvalidBlockHeight
	}

	types.SetLastAppliedBlockSource(blockMetaData.Source.String())

	m.logger.Debug("Applying block", "height", block.Header.Height, "source", blockMetaData.Source.String())

	// Check if the app's last block height is the same as the currently produced block height
	isBlockAlreadyApplied, err := m.isHeightAlreadyApplied(block.Header.Height)
	if err != nil {
		return fmt.Errorf("check if block is already applied: %w", err)
	}
	// In case the following true, it means we crashed after the commit and before updating the store height.
	// In that case we'll want to align the store with the app state and continue to the next block.
	if isBlockAlreadyApplied {
		// In this case, where the app was committed, but the state wasn't updated
		// it will update the state from appInfo, saved responses and validators.
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

	err = m.saveP2PBlockToBlockSync(block, commit)
	if err != nil {
		m.logger.Error("save block blocksync", "err", err)
	}

	m.logger.Info("executing block", "height", block.Header.Height)
	responses, err := m.Executor.ExecuteBlock(m.State, block)
	if err != nil {
		return fmt.Errorf("execute block: %w", err)
	}
	m.logger.Info("block executed", "height", block.Header.Height)

	dbBatch := m.Store.NewBatch()
	dbBatch, err = m.Store.SaveBlockResponses(block.Header.Height, responses, dbBatch)
	if err != nil {
		dbBatch.Discard()
		return fmt.Errorf("save block responses: %w", err)
	}

	// Get the validator changes from the app
	validators := m.State.NextValidators.Copy() // TODO: this will be changed when supporting multiple sequencers from the hub

	dbBatch, err = m.Store.SaveValidators(block.Header.Height, validators, dbBatch)
	if err != nil {
		dbBatch.Discard()
		return fmt.Errorf("save validators: %w", err)
	}

	err = dbBatch.Commit()
	if err != nil {
		return fmt.Errorf("commit batch to disk: %w", err)
	}
	m.logger.Info("committing block", "height", block.Header.Height)

	// Commit block to app
	appHash, retainHeight, err := m.Executor.Commit(m.State, block, responses)
	if err != nil {
		return fmt.Errorf("commit block: %w", err)
	}
	m.logger.Info("block committed", "height", block.Header.Height)

	// If failed here, after the app committed, but before the state is updated, we'll update the state on
	// UpdateStateFromApp using the saved responses and validators.

	// Update the state with the new app hash, last validators and store height from the commit.
	// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
	m.Executor.UpdateStateAfterCommit(m.State, responses, appHash, block.Header.Height, validators)
	_, err = m.Store.SaveState(m.State, nil)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}
	// Prune old heights, if requested by ABCI app.
	if 0 < retainHeight {
		select {
		case m.pruningC <- retainHeight:
		default:
			m.logger.Debug("pruning channel full. skipping pruning", "retainHeight", retainHeight)
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

func (m *Manager) attemptApplyCachedBlocks() error {
	m.logger.Info("onReceivedBlock locked")

	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	for {
		expectedHeight := m.State.NextHeight()

		cachedBlock, blockExists := m.blockCache.Get(expectedHeight)
		if !blockExists {
			break
		}
		if err := m.validateBlock(cachedBlock.Block, cachedBlock.Commit); err != nil {
			m.blockCache.Delete(cachedBlock.Block.Header.Height)
			// TODO: can we take an action here such as dropping the peer / reducing their reputation?
			return fmt.Errorf("block not valid at height %d, dropping it: err:%w", cachedBlock.Block.Header.Height, err)
		}

		err := m.applyBlock(cachedBlock.Block, cachedBlock.Commit, types.BlockMetaData{Source: cachedBlock.Source})
		if err != nil {
			return fmt.Errorf("apply cached block: expected height: %d: %w", expectedHeight, err)
		}
		m.logger.Info("Block applied", "height", expectedHeight)

		m.blockCache.Delete(cachedBlock.Block.Header.Height)
	}
	m.logger.Info("onReceivedBlock unlocked")

	return nil
}

func (m *Manager) validateBlock(block *types.Block, commit *types.Commit) error {
	// Currently we're assuming proposer is never nil as it's a pre-condition for
	// dymint to start
	proposer := m.SLClient.GetProposer()

	return types.ValidateProposedTransition(m.State, block, commit, proposer)
}
