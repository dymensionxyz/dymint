package block

import (
	"context"

	"cosmossdk.io/errors"
	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmtypes "github.com/tendermint/tendermint/types"
)

// applyBlock applies the block to the store and the abci app.
// steps: save block -> execute block with app -> update state -> commit block to app -> update store height and state hash.
// As the entire process can't be atomic we need to make sure the following condition apply before
// - block height is the expected block height on the store (height + 1).
// - block height is the expected block height on the app (last block height + 1).
func (m *Manager) applyBlock(ctx context.Context, block *types.Block, commit *types.Commit, blockMetaData blockMetaData) error {
	// TODO (#330): allow genesis block with height > 0 to be applied.
	// TODO: add switch case to have defined behavior for each case.
	//validate block height
	if block.Header.Height != m.store.NextHeight() {
		m.logger.Error("Block not applied. wrong height", "block height", block.Header.Height, "expected height", m.store.NextHeight())
		return types.ErrInvalidBlockHeight
	}

	m.logger.Debug("Applying block", "height", block.Header.Height, "source", blockMetaData.source)

	// Check if the app's last block height is the same as the currently produced block height
	isBlockAlreadyApplied, err := m.isHeightAlreadyApplied(block.Header.Height)
	if err != nil {
		return err
	}
	// In case the following true, it means we crashed after the commit and before updating the store height.
	// In that case we'll want to align the store with the app state and continue to the next block.
	if isBlockAlreadyApplied {
		err := m.UpdateStateFromApp()
		if err != nil {
			return err
		}
		m.logger.Debug("Aligned with app state required. Skipping to next block", "height", block.Header.Height)
		return nil
	}
	// Start applying the block assuming no inconsistency was found.
	_, err = m.store.SaveBlock(block, commit, nil)
	if err != nil {
		m.logger.Error("Failed to save block", "error", err)
		return err
	}

	responses, err := m.executeBlock(ctx, block, commit)
	if err != nil {
		m.logger.Error("Failed to execute block", "error", err)
		return err
	}

	newState, err := m.executor.UpdateStateFromResponses(responses, m.lastState, block)
	if err != nil {
		return err
	}

	batch := m.store.NewBatch()

	batch, err = m.store.SaveBlockResponses(block.Header.Height, responses, batch)
	if err != nil {
		batch.Discard()
		return err
	}

	m.lastState = newState
	batch, err = m.store.UpdateState(m.lastState, batch)
	if err != nil {
		batch.Discard()
		return err
	}
	batch, err = m.store.SaveValidators(block.Header.Height, m.lastState.Validators, batch)
	if err != nil {
		batch.Discard()
		return err
	}

	err = batch.Commit()
	if err != nil {
		m.logger.Error("Failed to persist batch to disk", "error", err)
		return err
	}

	// Commit block to app
	retainHeight, err := m.executor.Commit(ctx, &newState, block, responses)
	if err != nil {
		m.logger.Error("Failed to commit to the block", "error", err)
		return err
	}

	// Prune old heights, if requested by ABCI app.
	if retainHeight > 0 {
		pruned, err := m.pruneBlocks(retainHeight)
		if err != nil {
			m.logger.Error("failed to prune blocks", "retain_height", retainHeight, "err", err)
		} else {
			m.logger.Debug("pruned blocks", "pruned", pruned, "retain_height", retainHeight)
		}
	}

	// Update the state with the new app hash, last validators and store height from the commit.
	// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
	newState.LastValidators = m.lastState.Validators.Copy()
	newState.LastStoreHeight = block.Header.Height
	newState.BaseHeight = m.store.Base()

	_, err = m.store.UpdateState(newState, nil)
	if err != nil {
		m.logger.Error("Failed to update state", "error", err)
		return err
	}
	m.lastState = newState

	m.store.SetHeight(block.Header.Height)

	return nil
}

func (m *Manager) attemptApplyCachedBlocks(ctx context.Context) error {
	m.applyCachedBlockMutex.Lock()
	defer m.applyCachedBlockMutex.Unlock()

	for {
		expectedHeight := m.store.NextHeight()

		prevCachedBlock, blockExists := m.prevBlock[expectedHeight]
		prevCachedCommit, commitExists := m.prevCommit[expectedHeight]

		if !blockExists || !commitExists {
			break
		}

		m.logger.Debug("Applying cached block", "height", expectedHeight)
		err := m.applyBlock(ctx, prevCachedBlock, prevCachedCommit, blockMetaData{source: gossipedBlock})
		if err != nil {
			m.logger.Debug("Failed to apply previously cached block", "err", err)
			return err
		}
	}

	for k := range m.prevBlock {
		if k <= m.store.Height() {
			delete(m.prevBlock, k)
			delete(m.prevCommit, k)
		}
	}
	return nil
}

// isHeightAlreadyApplied checks if the block height is already applied to the app.
func (m *Manager) isHeightAlreadyApplied(blockHeight uint64) (bool, error) {
	proxyAppInfo, err := m.executor.GetAppInfo()
	if err != nil {
		return false, errors.Wrap(err, "get app info")
	}

	isBlockAlreadyApplied := uint64(proxyAppInfo.LastBlockHeight) == blockHeight

	// TODO: add switch case to validate better the current app state

	return isBlockAlreadyApplied, nil
}

// UpdateStateFromApp is responsible for aligning the state of the store from the abci app
func (m *Manager) UpdateStateFromApp() error {
	proxyAppInfo, err := m.executor.GetAppInfo()
	if err != nil {
		return errors.Wrap(err, "get app info")
	}

	appHeight := uint64(proxyAppInfo.LastBlockHeight)

	// update the state with the hash, last store height and last validators.
	m.lastState.AppHash = *(*[32]byte)(proxyAppInfo.LastBlockAppHash)
	m.lastState.LastStoreHeight = appHeight
	m.lastState.LastValidators = m.lastState.Validators.Copy()

	resp, err := m.store.LoadBlockResponses(appHeight)
	if err != nil {
		return errors.Wrap(err, "failed to load block responses")
	}
	copy(m.lastState.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	_, err = m.store.UpdateState(m.lastState, nil)
	if err != nil {
		return errors.Wrap(err, "failed to update state")
	}
	m.store.SetHeight(appHeight)
	return nil
}

func (m *Manager) executeBlock(ctx context.Context, block *types.Block, commit *types.Commit) (*tmstate.ABCIResponses, error) {
	// Currently we're assuming proposer is never nil as it's a pre-condition for
	// dymint to start
	proposer := m.settlementClient.GetProposer()

	if err := m.executor.Validate(m.lastState, block, commit, proposer); err != nil {
		return &tmstate.ABCIResponses{}, err
	}

	responses, err := m.executor.Execute(ctx, m.lastState, block)
	if err != nil {
		return &tmstate.ABCIResponses{}, err
	}

	return responses, nil
}

func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	gossipedBlock := p2p.GossipedBlock{Block: block, Commit: commit}
	gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
	if err != nil {
		m.logger.Error("Failed to marshal block", "error", err)
		return err
	}
	if err := m.p2pClient.GossipBlock(ctx, gossipedBlockBytes); err != nil {
		m.logger.Error("Failed to gossip block", "error", err)
		return err
	}
	return nil

}
