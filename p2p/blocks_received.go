package p2p

import "sync"


type BlocksReceived struct {
	blocksReceived   map[uint64]struct{}
	latestSeenHeight uint64
	
	blockReceivedMu sync.Mutex
}


func (br *BlocksReceived) AddBlockReceived(height uint64) {
	br.latestSeenHeight = max(height, br.latestSeenHeight)
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	br.blocksReceived[height] = struct{}{}
}


func (br *BlocksReceived) IsBlockReceived(height uint64) bool {
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	_, ok := br.blocksReceived[height]
	return ok
}


func (br *BlocksReceived) RemoveBlocksReceivedUpToHeight(appliedHeight uint64) {
	br.blockReceivedMu.Lock()
	defer br.blockReceivedMu.Unlock()
	for h := range br.blocksReceived {
		if h < appliedHeight {
			delete(br.blocksReceived, h)
		}
	}
}


func (br *BlocksReceived) GetLatestSeenHeight() uint64 {
	return br.latestSeenHeight
}
