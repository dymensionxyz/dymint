package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onReceivedBlock receives a block received event from P2P, saves the block to a cache and tries to apply the blocks from the cache.
func (m *Manager) onReceivedBlock(event pubsub.Message) {
	eventData, ok := event.Data().(p2p.P2PBlockEvent)
	if !ok {
		m.logger.Error("onReceivedBlock", "err", "wrong event data received")
		return
	}
	var source blockSource

	if len(event.Events()[p2p.EventTypeKey]) != 1 {
		m.logger.Error("onReceivedBlock", "err", "wrong number of event types received with the event", "received", len(event.Events()[p2p.EventTypeKey]))
		return
	}

	switch event.Events()[p2p.EventTypeKey][0] {
	case p2p.EventNewBlockSyncBlock:
		source = blocksyncBlock
	case p2p.EventNewGossipedBlock:
		source = gossipedBlock
	default:
		m.logger.Error("onReceivedBlock", "err", "wrong event type received", "type", event.Events()[p2p.EventTypeKey][0])
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

// gossipBlock sends created blocks by the sequencer to full-nodes using P2P gossipSub
func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	m.logger.Info("Gossipping block", "height", block.Header.Height)
	gossipedBlock := p2p.P2PBlockEvent{Block: block, Commit: commit}
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
