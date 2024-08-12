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
	eventData, ok := event.Data().(p2p.BlockData)
	if !ok {
		m.logger.Error("onReceivedBlock", "err", "wrong event data received")
		return
	}
	var source types.BlockSource

	if len(event.Events()[p2p.EventTypeKey]) != 1 {
		m.logger.Error("onReceivedBlock", "err", "wrong number of event types received with the event", "received", len(event.Events()[p2p.EventTypeKey]))
		return
	}

	switch event.Events()[p2p.EventTypeKey][0] {
	case p2p.EventNewBlockSyncBlock:
		source = types.BlockSync
	case p2p.EventNewGossipedBlock:
		source = types.Gossiped
	default:
		m.logger.Error("onReceivedBlock", "err", "wrong event type received", "type", event.Events()[p2p.EventTypeKey][0])
		return
	}

	block := eventData.Block
	commit := eventData.Commit
	height := block.Header.Height
	m.retrieverMu.Lock() // needed to protect blockCache access

	// It is not strictly necessary to return early, for correctness, but doing so helps us avoid mutex pressure and unnecessary repeated attempts to apply cached blocks
	if m.blockCache.Has(height) {
		m.retrieverMu.Unlock()
		return
	}

	m.UpdateTargetHeight(height)
	types.LastReceivedP2PHeightGauge.Set(float64(height))

	m.logger.Debug("Received new block via gossip.", "block height", height, "store height", m.State.Height(), "n cachedBlocks", m.blockCache.Size())

	nextHeight := m.State.NextHeight()
	if height >= nextHeight {
		m.blockCache.Add(height, &block, &commit, source)
	}
	m.retrieverMu.Unlock() // have to give this up as it's locked again in attempt apply, and we're not re-entrant

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("Applying cached blocks.", "err", err)
	}
}

// gossipBlock sends created blocks by the sequencer to full-nodes using P2P gossipSub
func (m *Manager) gossipBlock(ctx context.Context, block types.Block, commit types.Commit) error {
	m.logger.Info("Gossipping block", "height", block.Header.Height)
	gossipedBlock := p2p.BlockData{Block: block, Commit: commit}
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
