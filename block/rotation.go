package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/uuid"
)

func (m *Manager) MonitorSequencerRotation(ctx context.Context) []byte {
	sequencerRotationEventClient := fmt.Sprintf("%s-%s", "sequencer_rotation", uuid.New().String())
	subscription, err := m.Pubsub.Subscribe(ctx, sequencerRotationEventClient, settlement.EventQueryRotationStarted)
	if err != nil {
		panic("Error subscribing to events")
	}
	defer m.Pubsub.UnsubscribeAll(ctx, sequencerRotationEventClient)

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-subscription.Out():
			eventData, _ := event.Data().(*settlement.EventDataRotationStarted)
			nextSeq := eventData.NextSeqAddr
			m.logger.Info("Sequencer rotation started.", "next_seq", nextSeq)
			return []byte(nextSeq)
		}
	}
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
