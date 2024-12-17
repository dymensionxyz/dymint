package block

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	errorsmod "cosmossdk.io/errors"

	"github.com/dymensionxyz/dymint/types"
)

func (m *Manager) applyBlockWithFraudHandling(block *types.Block, commit *types.Commit, blockMetaData types.BlockMetaData) error {
	validateWithFraud := func() error {
		if err := m.validateBlockBeforeApply(block, commit); err != nil {
			m.blockCache.Delete(block.Header.Height)

			return fmt.Errorf("block not valid at height %d, dropping it: err:%w", block.Header.Height, err)
		}

		if err := m.applyBlock(block, commit, blockMetaData); err != nil {
			return fmt.Errorf("apply block: %w", err)
		}

		return nil
	}

	err := validateWithFraud()
	if errors.Is(err, gerrc.ErrFault) {
		m.FraudHandler.HandleFault(m.Ctx, err)
	}

	return err
}

func (m *Manager) applyBlock(block *types.Block, commit *types.Commit, blockMetaData types.BlockMetaData) error {
	var retainHeight int64

	if block.Header.Height != m.State.NextHeight() {
		return types.ErrInvalidBlockHeight
	}

	types.SetLastAppliedBlockSource(blockMetaData.Source.String())

	isBlockAlreadyApplied, err := m.isHeightAlreadyApplied(block.Header.Height)
	if err != nil {
		return fmt.Errorf("check if block is already applied: %w", err)
	}

	if isBlockAlreadyApplied {
		err := m.UpdateStateFromApp(block.Header.Hash())
		if err != nil {
			return fmt.Errorf("update state from app: %w", err)
		}
	} else {
		var appHash []byte

		_, err = m.Store.SaveBlock(block, commit, nil)
		if err != nil {
			return fmt.Errorf("save block: %w", err)
		}

		err := m.saveP2PBlockToBlockSync(block, commit)
		if err != nil {
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

		appHash, retainHeight, err = m.Executor.Commit(m.State, block, responses)
		if err != nil {
			return fmt.Errorf("commit block: %w", err)
		}

		if 0 < retainHeight {
			select {
			case m.pruningC <- retainHeight:
			default:
			}
		}

		m.Executor.UpdateStateAfterCommit(m.State, responses, appHash, block.Header.Height, block.Header.Hash())

	}

	m.LastBlockTime.Store(block.Header.GetTimestamp().UTC().UnixNano())

	proposer := m.State.GetProposer()
	if proposer == nil {
		return fmt.Errorf("logic error: got nil proposer while applying block")
	}

	batch := m.Store.NewBatch()

	batch, err = m.Store.SaveProposer(block.Header.Height, *proposer, batch)
	if err != nil {
		return fmt.Errorf("save proposer: %w", err)
	}

	isProposerUpdated := m.Executor.UpdateProposerFromBlock(m.State, m.Sequencers, block)

	batch, err = m.Store.SaveState(m.State, batch)
	if err != nil {
		return fmt.Errorf("update state: %w", err)
	}

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

	err = m.ValidateConfigWithRollappParams()
	if err != nil {
		return err
	}

	if isProposerUpdated && m.AmIProposerOnRollapp() {
		panic("I'm the new Proposer now. restarting as a proposer")
	}

	return nil
}

func (m *Manager) isHeightAlreadyApplied(blockHeight uint64) (bool, error) {
	proxyAppInfo, err := m.Executor.GetAppInfo()
	if err != nil {
		return false, errorsmod.Wrap(err, "get app info")
	}

	isBlockAlreadyApplied := uint64(proxyAppInfo.LastBlockHeight) == blockHeight

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
	}

	return nil
}

func (m *Manager) validateBlockBeforeApply(block *types.Block, commit *types.Commit) error {
	return types.ValidateProposedTransition(m.State, block, commit, m.State.GetProposerPubKey())
}
