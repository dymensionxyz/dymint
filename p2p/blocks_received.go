package p2p

import "sync"

// BlocksReceived tracks blocks received from P2P to know what are the missing blocks that need to be requested on demand
type BlocksReceived struct {
	blocksReceived   map[uint64]struct{}
	latestSeenHeight uint64
	// mutex to protect blocksReceived map access
	blockReceivedMu sync.Mutex
}

// AddBlockReceived adds the block height to a map
func (br *BlocksReceived) AddBlockReceived(height uint64) {
	br.latestSeenHeight = max(height, br.latestSeenHeight)
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	br.blocksReceived[height] = struct{}{}
}

// IsBlockReceived checks if a block height is already received
func (br *BlocksReceived) IsBlockReceived(height uint64) bool {
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	_, ok := br.blocksReceived[height]
	return ok
}

// RemoveBlocksReceivedUpToHeight clears previous received block heights
func (br *BlocksReceived) RemoveBlocksReceivedUpToHeight(appliedHeight uint64) {
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	for h := range br.blocksReceived {
		if h < appliedHeight {
			delete(br.blocksReceived, h)
		}
	}
}

// GetLatestSeenHeight returns the latest height stored
func (br *BlocksReceived) GetLatestSeenHeight() uint64 {
	return br.latestSeenHeight
}
