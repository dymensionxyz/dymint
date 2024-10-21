package block

import (
	"github.com/dymensionxyz/dymint/types"
)

type Cache struct {
	// concurrency managed by Manager.retrieverMu mutex
	cache          map[uint64]types.CachedBlock
	lastSeenHeight uint64
}

func (m *Cache) Add(h uint64, b *types.Block, c *types.Commit, source types.BlockSource) {
	m.cache[h] = types.CachedBlock{Block: b, Commit: c, Source: source}
	types.BlockCacheSizeGauge.Set(float64(m.Size()))

	if h > m.lastSeenHeight {
		m.lastSeenHeight = h
	}
}

func (m *Cache) Delete(h uint64) {
	delete(m.cache, h)
	types.BlockCacheSizeGauge.Set(float64(m.Size()))
}

func (m *Cache) Get(h uint64) (types.CachedBlock, bool) {
	ret, found := m.cache[h]
	return ret, found
}

func (m *Cache) Has(h uint64) bool {
	_, found := m.Get(h)
	return found
}

func (m *Cache) Size() int {
	return len(m.cache)
}

func (m *Cache) LastSeenHeight() uint64 {
	return m.lastSeenHeight
}
