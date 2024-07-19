package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onReceivedBlock will take a block and apply it
func (m *Manager) onReceivedBlock(event pubsub.Message) {
	eventData, ok := event.Data().(p2p.P2PBlock)
	if !ok {
		m.logger.Error("onReceivedBlock", "err", "wrong event data received")
		return
	}
	var source blockSource
	switch event.Events()[p2p.EventTypeKey][0] {
	case p2p.EventNewBlockSyncBlock:
		source = blocksyncBlock
	case p2p.EventNewGossipedBlock:
		source = gossipedBlock
	default:
		m.logger.Error("onReceivedBlock", "err", "wrong event type received")
		return
	}

	block := eventData.Block
	commit := eventData.Commit

	m.logger.Debug("Received new block.", "source", source, "block height", block.Header.Height, "store height", m.State.Height(), "n cachedBlocks", len(m.blockCache))

	ok = m.attemptCacheBlock(&block, &commit, source)
	if !ok {
		return
	}

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("Applying cached blocks.", "err", err)
	}
}

func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	m.logger.Info("Gossipping block", "height", block.Header.Height)
	gossipedBlock := p2p.P2PBlock{Block: block, Commit: commit}
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
