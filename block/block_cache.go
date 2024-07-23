package block

import (
	"sync"

	"github.com/dymensionxyz/dymint/types"
)

type Cache struct {
	cache map[uint64]CachedBlock
	sync.Mutex
}

func (m *Cache) AddBlockToCache(h uint64, b *types.Block, c *types.Commit) {
	m.Lock()
	defer m.Unlock()
	m.cache[h] = CachedBlock{Block: b, Commit: c}

	types.BlockCacheSize.Set(float64(len(m.cache)))
	types.LowestPendingBlockHeight.Set(m.GetLowestCachedBlockHeight())
	types.HighestReceivedBlockHeight.Set(m.GetHighestCachedBlockHeight())
}

func (m *Cache) DeleteBlockFromCache(h uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.cache, h)

	types.BlockCacheSize.Set(float64(len(m.cache)))
	types.LowestPendingBlockHeight.Set(float64(m.GetLowestCachedBlockHeight()))
	types.HighestReceivedBlockHeight.Set(float64(m.GetHighestCachedBlockHeight()))
}

func (m *Cache) GetLowestCachedBlockHeight() float64 {
	m.Lock()
	defer m.Unlock()
	var lowest uint64
	for h := range m.cache {
		if lowest == 0 || h < lowest {
			lowest = h
		}
	}
	return float64(lowest)
}

func (m *Cache) GetHighestCachedBlockHeight() float64 {
	m.Lock()
	defer m.Unlock()
	var highest uint64
	for h := range m.cache {
		if h > highest {
			highest = h
		}
	}
	return float64(highest)
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
