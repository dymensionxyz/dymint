package block

import (
	"github.com/dymensionxyz/dymint/types"
)

type Cache struct {
	// concurrency managed by Manager.retrieverMu mutex
	cache map[uint64]types.CachedBlock
}

func (m *Cache) AddBlockToCache(h uint64, b *types.Block, c *types.Commit) {
	m.cache[h] = types.CachedBlock{Block: b, Commit: c}
	types.BlockCacheSizeGauge.Set(float64(m.Size()))
}

func (m *Cache) DeleteBlockFromCache(h uint64) {
	delete(m.cache, h)
	types.BlockCacheSizeGauge.Set(float64(m.Size()))
}

func (m *Cache) GetBlockFromCache(h uint64) (types.CachedBlock, bool) {
	ret, found := m.cache[h]
	return ret, found
}

func (m *Cache) HasBlockInCache(h uint64) bool {
	_, found := m.GetBlockFromCache(h)
	return found
}

func (m *Cache) Size() int {
	return len(m.cache)
}
