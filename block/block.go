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
	var retainHeight int64

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
		m.logger.Info("updated state from app commit", "height", block.Header.Height)
	} else {
		var appHash []byte
		// Start applying the block assuming no inconsistency was found.
		_, err = m.Store.SaveBlock(block, commit, nil)
		if err != nil {
			return fmt.Errorf("save block: %w", err)
		}

		responses, err := m.Executor.ExecuteBlock(m.State, block)
		if err != nil {
			return fmt.Errorf("execute block: %w", err)
		}

		_, err = m.Store.SaveBlockResponses(block.Header.Height, responses, nil)
		if err != nil {
			return fmt.Errorf("save block responses: %w", err)
		}

		// Commit block to app
		appHash, retainHeight, err = m.Executor.Commit(m.State, block, responses)
		if err != nil {
			return fmt.Errorf("commit block: %w", err)
		}

		// If failed here, after the app committed, but before the state is updated, we'll update the state on
		// UpdateStateFromApp using the saved responses

		// Update the state with the new app hash, and store height from the commit.
		// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
		m.Executor.UpdateStateAfterCommit(m.State, responses, appHash, block.Header.Height)
	}

	// update validators to state from block
	m.Executor.UpdateProposerFromBlock(m.State, block)

	// save validators to store to be queried over RPC
	batch := m.Store.NewBatch()
	batch, err = m.Store.SaveValidators(block.Header.Height, &m.State.Sequencers, batch)
	if err != nil {
		return fmt.Errorf("save validators: %w", err)
	}

	batch, err = m.Store.SaveState(m.State, batch)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	err = batch.Commit()
	if err != nil {
		return fmt.Errorf("commit state: %w", err)
	}

	types.RollappHeightGauge.Set(float64(block.Header.Height))

	// Prune old heights, if requested by ABCI app.
	if 0 < retainHeight {
		err = m.pruneBlocks(uint64(retainHeight))
		if err != nil {
			m.logger.Error("prune blocks", "retain_height", retainHeight, "err", err)
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
	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	for {
		expectedHeight := m.State.NextHeight()

		cachedBlock, blockExists := m.blockCache.GetBlockFromCache(expectedHeight)
		if !blockExists {
			break
		}
		if err := m.validateBlockBeforeApply(cachedBlock.Block, cachedBlock.Commit); err != nil {
			m.blockCache.DeleteBlockFromCache(cachedBlock.Block.Header.Height)
			// TODO: can we take an action here such as dropping the peer / reducing their reputation?
			return fmt.Errorf("block not valid at height %d, dropping it: err:%w", cachedBlock.Block.Header.Height, err)
		}

		err := m.applyBlock(cachedBlock.Block, cachedBlock.Commit, types.BlockMetaData{Source: types.GossipedBlock})
		if err != nil {
			return fmt.Errorf("apply cached block: expected height: %d: %w", expectedHeight, err)
		}
		m.logger.Info("Block applied", "height", expectedHeight)

		m.blockCache.DeleteBlockFromCache(cachedBlock.Block.Header.Height)
	}

	return nil
}

// This function validates the block and commit against the state before applying it.
func (m *Manager) validateBlockBeforeApply(block *types.Block, commit *types.Commit) error {
	if err := block.ValidateWithState(m.State); err != nil {
		return fmt.Errorf("block: %w", err)
	}

	if err := commit.ValidateWithHeader(m.GetProposerPubKey(), &block.Header); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
