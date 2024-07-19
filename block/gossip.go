package block

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
)

// onNewGossipedBlock will take a block and apply it
func (m *Manager) onNewGossipedBlock(event pubsub.Message) {
	eventData, _ := event.Data().(p2p.GossipedBlock)
	block := eventData.Block
	commit := eventData.Commit
	height := block.Header.Height

	m.LastReceivedP2PHeight.Store(height)

	if m.HasBlockInCache(height) {
		return
	}

	m.logger.Debug("Received new block via gossip.", "block height", height, "store height", m.State.Height(), "n cachedBlocks", m.BlockCacheSize.Load())

	nextHeight := m.State.NextHeight()
	if height >= nextHeight {
		m.AddBlockToCache(height, &block, &commit)
	}

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("Applying cached blocks.", "err", err)
	}
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
