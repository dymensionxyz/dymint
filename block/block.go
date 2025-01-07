package block

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"

	"github.com/dymensionxyz/dymint/types"
)

// applyBlockWithFraudHandling calls applyBlock and validateBlockBeforeApply with fraud handling.
func (m *Manager) applyBlockWithFraudHandling(block *types.Block, commit *types.Commit, blockMetaData types.BlockMetaData) error {
	if m.Conf.SkipValidationHeight != block.Header.Height {
		if err := m.validateBlockBeforeApply(block, commit); err != nil {
			m.blockCache.Delete(block.Header.Height)
			// TODO: can we take an action here such as dropping the peer / reducing their reputation?
			return fmt.Errorf("block not valid at height %d, dropping it: err:%w", block.Header.Height, err)
		}
	}

	if err := m.applyBlock(block, commit, blockMetaData); err != nil {
		return fmt.Errorf("apply block: %w", err)
	}

	return nil
}

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
	// In case the following true, it means we crashed after the app commit but before updating the state
	// In that case we'll want to align the state with the app commit result, as if the block was applied.
	if isBlockAlreadyApplied {
		err := m.UpdateStateFromApp(block)
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

		err := m.saveP2PBlockToBlockSync(block, commit)
		if err != nil {
			m.logger.Error("save block blocksync", "err", err)
		}

		responses, err := m.Executor.ExecuteBlock(block)
		if err != nil {
			return fmt.Errorf("execute block: %w", err)
		}

		_, err = m.Store.SaveBlockResponses(block.Header.Height, responses, nil)
		if err != nil {
			return fmt.Errorf("save block responses: %w", err)
		}

		_, err = m.Store.SaveBlockSource(block.Header.Height, blockMetaData.Source, nil)
		if err != nil {
			return fmt.Errorf("save block source: %w", err)
		}

		_, err = m.Store.SaveDRSVersion(block.Header.Height, responses.EndBlock.RollappParamUpdates.DrsVersion, nil)
		if err != nil {
			return fmt.Errorf("add drs version: %w", err)
		}

		// Commit block to app
		appHash, retainHeight, err = m.Executor.Commit(m.State, block, responses)
		if err != nil {
			return fmt.Errorf("commit block: %w", err)
		}

		// Prune old heights, if requested by ABCI app.
		// retainHeight is determined by currentHeight - min-retain-blocks (app.toml config).
		// Unless max_age_num_blocks in consensus params is higher than min-retain-block, then max_age_num_blocks will be used instead of min-retain-blocks.

		if 0 < retainHeight {
			select {
			case m.pruningC <- retainHeight:
			default:
				m.logger.Debug("pruning channel full. skipping pruning", "retainHeight", retainHeight)
			}
		}
		// Update the state with the new app hash, and store height from the commit.
		// Every one of those, if happens before commit, prevents us from re-executing the block in case failed during commit.
		m.Executor.UpdateStateAfterCommit(m.State, responses, appHash, block)

	}

	// Update the store:
	//  1. Save the proposer for the current height to the store.
	//  2. Update the proposer in the state in case of rotation.
	//  3. Save the state to the store (independently of the height). Here the proposer might differ from (1).
	//  4. Save the last block sequencer set to the store if it's present (only applicable in the sequencer mode).
	// here, (3) helps properly handle reboots (specifically when there's rotation).
	// If reboot happens after block H (which rotates seqA -> seqB):
	//  - Block H+1 will be signed by seqB.
	//  - The state must have seqB as proposer.

	// Proposer cannot be empty while applying the block
	proposer := m.State.GetProposer()
	if proposer == nil {
		return fmt.Errorf("logic error: got nil proposer while applying block")
	}

	batch := m.Store.NewBatch()

	// 1. Save the proposer for the current height to the store.
	// Proposer in the store is used for RPC queries.
	batch, err = m.Store.SaveProposer(block.Header.Height, *proposer, batch)
	if err != nil {
		return fmt.Errorf("save proposer: %w", err)
	}

	// 2. Update the proposer in the state in case of rotation happened on the rollapp level (not necessarily on the hub yet).
	isProposerUpdated := m.Executor.UpdateProposerFromBlock(m.State, m.Sequencers, block)

	// 3. Save the state to the store (independently of the height). Here the proposer might differ from (1).
	batch, err = m.Store.SaveState(m.State, batch)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	// 4. Save the last block sequencer set to the store if it's present (only applicable in the sequencer mode).
	// The set from the state is dumped to memory on reboots. It helps to avoid sending unnecessary
	// UspertSequencer consensus messages on reboots. This is not a 100% solution, because the sequencer set
	// is not persisted in the store in full node mode. It's only used in the proposer mode. Therefore,
	// on rotation from the full node to the proposer, the sequencer set is duplicated as consensus msgs.
	// Though single-time duplication it's not a big deal.
	if len(blockMetaData.SequencerSet) != 0 {
		batch, err = m.Store.SaveLastBlockSequencerSet(blockMetaData.SequencerSet, batch)
		if err != nil {
			return fmt.Errorf("save last block sequencer set: %w", err)
		}
	}

	err = batch.Commit()
	if err != nil {
		return fmt.Errorf("commit state: %w", err)
	}

	types.RollappHeightGauge.Set(float64(block.Header.Height))

	m.blockCache.Delete(block.Header.Height)

	// validate whether configuration params and rollapp consensus params keep in line, after rollapp params are updated from the responses received in the block execution
	err = m.ValidateConfigWithRollappParams()
	if err != nil {
		return err
	}

	// Check if there was an Update for the proposer and if I am the new proposer.
	// If so, restart so I can start as the proposer.
	// For current proposer, we don't want to restart because we still need to send the last batch.
	// This will be done as part of the `rotate` function.
	if isProposerUpdated && m.AmIProposerOnRollapp() {
		panic("I'm the new Proposer now. restarting as a proposer")
	}

	return nil
}

// isHeightAlreadyApplied checks if the block height is already applied to the app.
func (m *Manager) isHeightAlreadyApplied(blockHeight uint64) (bool, error) {
	proxyAppInfo, err := m.Executor.GetAppInfo()
	if err != nil {
		return false, errorsmod.Wrap(err, "get app info")
	}

	isBlockAlreadyApplied := uint64(proxyAppInfo.LastBlockHeight) == blockHeight //nolint:gosec // LastBlockHeight is always positive

	// TODO: add switch case to validate better the current app state

	return isBlockAlreadyApplied, nil
}

func (m *Manager) attemptApplyCachedBlocks() error {
	m.retrieverMu.Lock()
	defer m.retrieverMu.Unlock()

	for {
		expectedHeight := m.State.NextHeight()

		cachedBlock, blockExists := m.blockCache.Get(expectedHeight)
		if !blockExists {
			break
		}
		if cachedBlock.Block.GetRevision() != m.State.GetRevision() {
			break
		}
		err := m.applyBlockWithFraudHandling(cachedBlock.Block, cachedBlock.Commit, types.BlockMetaData{Source: cachedBlock.Source})
		if err != nil {
			return fmt.Errorf("apply cached block: expected height: %d: %w", expectedHeight, err)
		}
		m.logger.Info("Block applied", "height", expectedHeight)
	}

	return nil
}

// This function validates the block and commit against the state before applying it.
func (m *Manager) validateBlockBeforeApply(block *types.Block, commit *types.Commit) error {
	return types.ValidateProposedTransition(m.State, block, commit, m.State.GetProposerPubKey())
}
