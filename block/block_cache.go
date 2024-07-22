package block

import (
	"sync"

	"github.com/dymensionxyz/dymint/types"
)

type Cache struct {
	cache map[uint64]CachedBlock
	sync.Mutex
	lowestCachedBlockHeight  uint64
	highestCachedBlockHeight uint64
}

func (m *Cache) AddBlockToCache(h uint64, b *types.Block, c *types.Commit) {
	m.Lock()
	defer m.Unlock()
	m.cache[h] = CachedBlock{Block: b, Commit: c}

	types.NumberOfAccumulatedBlocks.Set(float64(len(m.cache)))
	if m.lowestCachedBlockHeight == 0 || h < m.lowestCachedBlockHeight {
		m.lowestCachedBlockHeight = h
		types.LowestPendingBlockHeight.Set(float64(h))
	}
	if h > m.highestCachedBlockHeight {
		m.highestCachedBlockHeight = h
		types.HighestReceivedBlockHeight.Set(float64(h))
	}
}

func (m *Cache) DeleteBlockFromCache(h uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.cache, h)

	types.NumberOfAccumulatedBlocks.Set(float64(len(m.cache)))
	if h == m.lowestCachedBlockHeight {
		m.lowestCachedBlockHeight = h + 1
		types.LowestPendingBlockHeight.Set(float64(m.lowestCachedBlockHeight))
	}
	if h == m.highestCachedBlockHeight {
		m.highestCachedBlockHeight = h - 1
		types.HighestReceivedBlockHeight.Set(float64(m.highestCachedBlockHeight))
	}
}

func (m *Cache) GetBlockFromCache(h uint64) (CachedBlock, bool) {
	m.Lock()
	defer m.Unlock()
	ret, found := m.cache[h]
	return ret, found
}

func (m *Cache) HasBlockInCache(h uint64) bool {
	_, found := m.GetBlockFromCache(h)
	return found
}

func (m *Cache) Size() int {
	m.Lock()
	defer m.Unlock()
	return len(m.cache)
}
