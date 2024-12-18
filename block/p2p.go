package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

func (m *Manager) OnReceivedBlock(event pubsub.Message) {
	eventData, ok := event.Data().(p2p.BlockData)
	if !ok {
		return // when can it happen
	}
	var source types.BlockSource

	if len(event.Events()[p2p.EventTypeKey]) != 1 {
		return
	}

	switch event.Events()[p2p.EventTypeKey][0] {
	case p2p.EventNewBlockSyncBlock:
		source = types.BlockSync
	case p2p.EventNewGossipedBlock:
		source = types.Gossiped
	default:
		return
	}

	block := eventData.Block
	commit := eventData.Commit
	height := block.Header.Height

	if block.Header.Height < m.State.NextHeight() {
		return
	}
	m.retrieverMu.Lock()

	if m.blockCache.Has(height) {
		m.retrieverMu.Unlock()
		return
	}

	m.UpdateTargetHeight(height)
	types.LastReceivedP2PHeightGauge.Set(float64(height)) // this is wrong, it can go backwards

	m.blockCache.Add(height, &block, &commit, source)

	m.retrieverMu.Unlock()

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.freezeNode(err)
	}
}

func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	gossipedBlock := p2p.BlockData{Block: block, Commit: commit}
	gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal binary: %w: %w", err, ErrNonRecoverable)
	}
	if err := m.P2PClient.GossipBlock(ctx, gossipedBlockBytes); err != nil {
		return fmt.Errorf("p2p gossip block: %w: %w", err, ErrRecoverable)
	}

	return nil
}

func (m *Manager) saveP2PBlockToBlockSync(block *types.Block, commit *types.Commit) error {
	gossipedBlock := p2p.BlockData{Block: *block, Commit: *commit}
	gossipedBlockBytes, err := gossipedBlock.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshal binary: %w: %w", err, ErrNonRecoverable)
	}
	err = m.P2PClient.SaveBlock(context.Background(), block.Header.Height, block.GetRevision(), gossipedBlockBytes)
	if err != nil {
	}
	return nil
}
