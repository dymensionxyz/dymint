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
	m.retrieverMu.Lock() // needed to protect blockCache access
	height := block.Header.Height
	// It is not strictly necessary to return early, for correctness, but doing so helps us avoid mutex pressure and unnecessary repeated attempts to apply cached blocks
	if m.blockCache.HasBlockInCache(height) {
		m.retrieverMu.Unlock()
		return
	}

	m.UpdateTargetHeight(height)
	types.LastReceivedP2PHeightGauge.Set(float64(height))

	m.logger.Debug("Received new block via gossip.", "block height", height, "store height", m.State.Height(), "n cachedBlocks", m.blockCache.Size())

	nextHeight := m.State.NextHeight()
	if height >= nextHeight {
		m.blockCache.AddBlockToCache(height, &block, &commit)
	}
	m.retrieverMu.Unlock() // have to give this up as it's locked again in attempt apply, and we're not re-entrant

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
