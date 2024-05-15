package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewGossippedBlock will take a block and apply it
func (m *Manager) onNewGossipedBlock(event pubsub.Message) {
	eventData := event.Data().(p2p.GossipedBlock)
	block := eventData.Block
	commit := eventData.Commit
	m.retrieverMutex.Lock() // needed to protect blockCache access
	_, found := m.blockCache[block.Header.Height]
	// It is not strictly necessary to return early, for correctness, but doing so helps us avoid mutex pressure and unnecessary repeated attempts to apply cached blocks
	if found {
		m.retrieverMutex.Unlock()
		return
	}

	m.logger.Debug("Received new block via gossip", "block height", block.Header.Height, "store height", m.State.Height(), "n cachedBlocks", len(m.blockCache))

	nextHeight := m.State.NextHeight()
	if block.Header.Height >= nextHeight {
		m.blockCache[block.Header.Height] = CachedBlock{
			Block:  &block,
			Commit: &commit,
		}
	}
	m.retrieverMutex.Unlock() // have to give this up as it's locked again in attempt apply, and we're not re-entrant

	err := m.attemptApplyCachedBlocks()
	if err != nil {
		m.logger.Error("applying cached blocks", "err", err)
	}
}

func (m *Manager) attemptApplyCachedBlocks() error {
	m.retrieverMutex.Lock()
	defer m.retrieverMutex.Unlock()

	for {
		expectedHeight := m.State.NextHeight()

		cachedBlock, blockExists := m.blockCache[expectedHeight]
		if !blockExists {
			break
		}
		if err := m.validateBlock(cachedBlock.Block, cachedBlock.Commit); err != nil {
			delete(m.blockCache, cachedBlock.Block.Header.Height)
			/// TODO: can we take an action here such as dropping the peer / reducing their reputation?
			return fmt.Errorf("block not valid at height %d, dropping it: err:%w", cachedBlock.Block.Header.Height, err)
		}

		err := m.applyBlock(cachedBlock.Block, cachedBlock.Commit, blockMetaData{source: gossipedBlock})
		if err != nil {
			return fmt.Errorf("apply cached block: expected height: %d: %w", expectedHeight, err)
		}
		m.logger.Debug("applied cached block", "height", expectedHeight)

		delete(m.blockCache, cachedBlock.Block.Header.Height)
	}

	return nil
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
