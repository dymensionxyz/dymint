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

	types.BlockCacheSize.Set(float64(len(m.cache)))

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

	types.BlockCacheSize.Set(float64(len(m.cache)))

	if h == m.lowestCachedBlockHeight {
		m.lowestCachedBlockHeight = m.findNextHeight(h)
		types.LowestPendingBlockHeight.Set(float64(m.lowestCachedBlockHeight))
	}
	if h == m.highestCachedBlockHeight {
		m.highestCachedBlockHeight = m.findPreviousHeight(h)
		types.HighestReceivedBlockHeight.Set(float64(m.highestCachedBlockHeight))
	}
}

func (m *Cache) findNextHeight(h uint64) uint64 {
	m.Lock()
	defer m.Unlock()
	for n := h; n <= m.highestCachedBlockHeight; n++ {
		if _, found := m.cache[n]; found {
			return n
		}
	}
	return 0
}

func (m *Cache) findPreviousHeight(h uint64) uint64 {
	m.Lock()
	defer m.Unlock()
	for n := h; n >= m.lowestCachedBlockHeight; n-- {
		if _, found := m.cache[n]; found {
			return n
		}
	}
	return 0
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
