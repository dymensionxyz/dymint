package block

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// applyBlock applies the block to the store and the abci app.
// Contract: block and commit must be validated before calling this function!
// steps: save block -> execute block with app -> update state -> commit block to app -> update store height and state hash.
// As the entire process can't be atomic we need to make sure the following condition apply before
// - block height is the expected block height on the store (height + 1).
// - block height is the expected block height on the app (last block height + 1).
func (m *Manager) applyBlock(block *types.Block, commit *types.Commit, blockMetaData blockMetaData) error {
	// TODO (#330): allow genesis block with height > 0 to be applied.
	// TODO: add switch case to have defined behavior for each case.
	// validate block height
	if block.Header.Height != m.store.NextHeight() {
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
	_, err = m.store.SaveBlock(block, commit, nil)
	if err != nil {
		return fmt.Errorf("save block: %w", err)
	}

	responses, err := m.executor.ExecuteBlock(m.lastState, block)
	if err != nil {
		return fmt.Errorf("execute block: %w", err)
	}

	newState, err := m.executor.UpdateStateFromResponses(responses, m.lastState, block)
	if err != nil {
		return fmt.Errorf("update state from responses: %w", err)
	}

	batch := m.store.NewBatch()

	batch, err = m.store.SaveBlockResponses(block.Header.Height, responses, batch)
	if err != nil {
		batch.Discard()
		return fmt.Errorf("save block responses: %w", err)
	}

	m.lastState = newState
	batch, err = m.store.UpdateState(m.lastState, batch)
	if err != nil {
		batch.Discard()
		return fmt.Errorf("update state: %w", err)
	}
	batch, err = m.store.SaveValidators(block.Header.Height, m.lastState.Validators, batch)
	if err != nil {
		batch.Discard()
		return fmt.Errorf("save validators: %w", err)
	}

	err = batch.Commit()
	if err != nil {
		return fmt.Errorf("commit batch to disk: %w", err)
	}

	// Commit block to app
	retainHeight, err := m.executor.Commit(&newState, block, responses)
	if err != nil {
		return fmt.Errorf("commit block: %w", err)
	}

	// Prune old heights, if requested by ABCI app.
	if retainHeight > 0 {
		pruned, err := m.pruneBlocks(retainHeight)
		if err != nil {
			m.logger.Error("prune blocks", "retain_height", retainHeight, "err", err)
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
		return fmt.Errorf("final update state: %w", err)
	}
	m.lastState = newState

	if ok := m.store.SetHeight(block.Header.Height); !ok {
		return fmt.Errorf("store set height: %d", block.Header.Height)
	}

	return nil
}

// TODO: move to gossip.go
func (m *Manager) attemptApplyCachedBlocks() error {
	m.retrieverMutex.Lock()
	defer m.retrieverMutex.Unlock()

	for {
		expectedHeight := m.store.NextHeight()

		cachedBlock, blockExists := m.blockCache[expectedHeight]
		if !blockExists {
			break
		}

		// Note: cached <block,commit> pairs have passed basic validation, so no need to validate again
		err := m.applyBlock(cachedBlock.Block, cachedBlock.Commit, blockMetaData{source: gossipedBlock})
		if err != nil {
			return fmt.Errorf("apply cached block: expected height: %d: %w", expectedHeight, err)
		}
		m.logger.Debug("applied cached block", "height", expectedHeight)

		delete(m.blockCache, cachedBlock.Block.Header.Height)
	}

	return nil
}

// isHeightAlreadyApplied checks if the block height is already applied to the app.
func (m *Manager) isHeightAlreadyApplied(blockHeight uint64) (bool, error) {
	proxyAppInfo, err := m.executor.GetAppInfo()
	if err != nil {
		return false, errorsmod.Wrap(err, "get app info")
	}

	isBlockAlreadyApplied := uint64(proxyAppInfo.LastBlockHeight) == blockHeight

	// TODO: add switch case to validate better the current app state

	return isBlockAlreadyApplied, nil
}

// UpdateStateFromApp is responsible for aligning the state of the store from the abci app
func (m *Manager) UpdateStateFromApp() error {
	proxyAppInfo, err := m.executor.GetAppInfo()
	if err != nil {
		return errorsmod.Wrap(err, "get app info")
	}

	appHeight := uint64(proxyAppInfo.LastBlockHeight)

	// update the state with the hash, last store height and last validators.
	m.lastState.AppHash = *(*[32]byte)(proxyAppInfo.LastBlockAppHash)
	m.lastState.LastStoreHeight = appHeight
	m.lastState.LastValidators = m.lastState.Validators.Copy()

	resp, err := m.store.LoadBlockResponses(appHeight)
	if err != nil {
		return errorsmod.Wrap(err, "load block responses")
	}
	copy(m.lastState.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	_, err = m.store.UpdateState(m.lastState, nil)
	if err != nil {
		return errorsmod.Wrap(err, "update state")
	}
	if ok := m.store.SetHeight(appHeight); !ok {
		return fmt.Errorf("store set height: %d", appHeight)
	}
	return nil
}

func (m *Manager) validateBlock(block *types.Block, commit *types.Commit) error {
	// Currently we're assuming proposer is never nil as it's a pre-condition for
	// dymint to start
	proposer := m.settlementClient.GetProposer()

	return types.ValidateProposedTransition(m.lastState, block, commit, proposer)
}

func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	gossipedBlock := p2p.GossipedBlock{Block: block, Commit: commit}
	gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal binary: %w: %w", err, ErrNonRecoverable)
	}
	if err := m.p2pClient.GossipBlock(ctx, gossipedBlockBytes); err != nil {
		// Although this boils down to publishing on a topic, we don't want to speculate too much on what
		// could cause that to fail, so we assume recoverable.
		return fmt.Errorf("p2p gossip block: %w: %w", err, ErrRecoverable)
	}
	return nil
}
