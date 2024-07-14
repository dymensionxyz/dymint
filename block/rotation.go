package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

func (m *Manager) MonitorSequencerRotation(ctx context.Context) []byte {
	// var subscription *pubsub.Subscription
	// subscription, err := m.Pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	// if err != nil {
	// m.logger.Error("subscribe to state update events", "error", err)
	// return err
	// }

	// for {
	// 	select {
	// 	//FIXME: listen to channel coming from state update response
	// 	case <-ctx.Done():
	// 		return nil
	// 	case event := <-subscription.Out():
	// 		eventData, _ := event.Data().(*settlement.EventDataNewBatchAccepted)
	// 		h := eventData.EndHeight

	// 		if h <= m.State.Height() {
	// 			m.logger.Debug(
	// 				"syncTargetLoop: received new settlement batch accepted with batch end height <= current store height, skipping.",
	// 				"target sync height (batch end height)",
	// 				h,
	// 				"current store height",
	// 				m.State.Height(),
	// 			)
	// 			continue
	// 		}
	// 		types.RollappHubHeightGauge.Set(float64(h))
	// 		m.targetSyncHeight.Set(diodes.GenericDataType(&h))
	// 		m.logger.Info("Set new target sync height", "height", h)
	// 	case <-subscription.Cancelled():
	// 		return fmt.Errorf("syncTargetLoop subscription canceled")
	// 	}
	// }
	return []byte{}
}

func (m *Manager) GetPreviousBlockHashes(forHeight uint64) (lastHeaderHash [32]byte, lastCommit *types.Commit, err error) {
	lastHeaderHash, lastCommit, err = loadPrevBlock(m.Store, forHeight-1)
	if err != nil {
		if !m.State.IsGenesis() { // allow prevBlock not to be found only on genesis
			return [32]byte{}, nil, fmt.Errorf("load prev block: %w: %w", err, ErrNonRecoverable)
		}
		lastHeaderHash = [32]byte{}
		lastCommit = &types.Commit{}
	}
	return lastHeaderHash, lastCommit, nil
}

func (m *Manager) produceLastBlock(nextSeqAddrHash []byte) (*types.Block, *types.Commit, error) {
	newHeight := m.State.NextHeight()
	lastHeaderHash, lastCommit, err := m.GetPreviousBlockHashes(newHeight)
	if err != nil {
		return nil, nil, fmt.Errorf("load prev block: %w", err)
	}

	var block *types.Block
	var commit *types.Commit
	// Check if there's an already stored block and commit at a newer height
	// If there is use that instead of creating a new block
	pendingBlock, err := m.Store.LoadBlock(newHeight)
	if err == nil {
		// Using an existing block
		block = pendingBlock
		commit, err = m.Store.LoadCommit(newHeight)
		if err != nil {
			return nil, nil, fmt.Errorf("load commit after load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
		}
		m.logger.Info("Using pending block.", "height", newHeight)
	} else if !errors.Is(err, gerrc.ErrNotFound) {
		return nil, nil, fmt.Errorf("load block: height: %d: %w: %w", newHeight, err, ErrNonRecoverable)
	} else {
		// Create a new block
		block, commit, err = m.createLastBlock(newHeight, lastCommit, lastHeaderHash, nextSeqAddrHash)
		if err != nil {
			return nil, nil, fmt.Errorf("create new block: %w", err)
		}
	}

	m.logger.Info("Last block created.", "height", newHeight, "next_seq_hash", nextSeqAddrHash)
	types.RollappBlockSizeBytesGauge.Set(float64(len(block.Data.Txs)))
	types.RollappBlockSizeTxsGauge.Set(float64(len(block.Data.Txs)))
	return block, commit, nil
}

func (m *Manager) CreateAndPostLastBatch(ctx context.Context, nextSeq []byte) error {
	// validate nextSeq is in the bonded set
	_, val := m.State.ActiveSequencer.BondedSet.GetByAddress(nextSeq)
	if val == nil {
		return fmt.Errorf("next sequencer not found in bonded set")
	}
	nextSeqHash := types.GetHash(val)

	//FIXME: submit all data accumulated thus far

	_, _, err := m.ProduceApplyGossipLastBlock(ctx, nextSeqHash)
	if errors.Is(err, context.Canceled) {
		m.logger.Error("Produce and gossip: context canceled.", "error", err)
		return nil
	}

	// FIXME: submit batch
	return err
}
