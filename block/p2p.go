package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	"github.com/ipfs/go-cid"
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

	m.logger.Debug("Received new block.", "via", source, "block height", block.Header.Height, "store height", m.State.Height(), "n cachedBlocks", len(m.blockCache))

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

	// adds the block to be used by block-sync protocol
	err = m.p2pClient.AddBlock(ctx, block.Header.Height, gossipedBlockBytes)
	if err != nil {
		m.logger.Error("adding block to p2p store: %w", err)
	}

	return nil
}

// content identifiers are re-advertised on node startup to make sure ids are always found in the network
func (m *Manager) refreshBlockSyncAdvertiseBlocks(ctx context.Context) {
	for h := uint64(1); h <= m.State.Height(); h++ {

		id, err := m.p2pClient.GetBlockId(ctx, h)
		if err == nil && id != cid.Undef {
			continue
		}
		id, err = m.Store.LoadBlockCid(h)
		if err != nil || id == cid.Undef {
			continue
		}

		err = m.p2pClient.AdvertiseBlock(ctx, h, id)
		if err == nil {
			continue
		}

	}
}
