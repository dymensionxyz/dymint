package block

import (
	"sync"

	"github.com/dymensionxyz/dymint/types"
)

type Cache struct {
	cache map[uint64]types.CachedBlock
	sync.Mutex
}

func (m *Cache) AddBlockToCache(h uint64, b *types.Block, c *types.Commit) {
	m.Lock()
	defer m.Unlock()
	m.cache[h] = types.CachedBlock{Block: b, Commit: c}
	types.BlockCacheSizeGauge.Set(float64(len(m.cache)))
}

func (m *Cache) DeleteBlockFromCache(h uint64) {
	m.Lock()
	defer m.Unlock()
	delete(m.cache, h)
	size := len(m.cache)
	types.BlockCacheSizeGauge.Set(float64(size))
}

func (m *Cache) GetBlockFromCache(h uint64) (types.CachedBlock, bool) {
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
