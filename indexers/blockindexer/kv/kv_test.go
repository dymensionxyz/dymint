package kv_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/types"
	"golang.org/x/exp/rand"

	blockidxkv "github.com/dymensionxyz/dymint/indexers/blockindexer/kv"
	"github.com/dymensionxyz/dymint/store"
)

func TestBlockIndexer(t *testing.T) {
	prefixStore := store.NewPrefixKV(store.NewDefaultInMemoryKVStore(), []byte("block_events"))
	indexer := blockidxkv.New(prefixStore)

	require.NoError(t, indexer.Index(types.EventDataNewBlockHeader{
		Header: types.Header{Height: 1},
		ResultBeginBlock: abci.ResponseBeginBlock{
			Events: []abci.Event{
				{
					Type: "begin_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("proposer"),
							Value: []byte("FCAA001"),
							Index: true,
						},
					},
				},
			},
		},
		ResultEndBlock: abci.ResponseEndBlock{
			Events: []abci.Event{
				{
					Type: "end_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("foo"),
							Value: []byte("100"),
							Index: true,
						},
					},
				},
			},
		},
	}))

	for i := 2; i < 12; i++ {
		var index bool
		if i%2 == 0 {
			index = true
		}

		require.NoError(t, indexer.Index(types.EventDataNewBlockHeader{
			Header: types.Header{Height: int64(i)},
			ResultBeginBlock: abci.ResponseBeginBlock{
				Events: []abci.Event{
					{
						Type: "begin_event",
						Attributes: []abci.EventAttribute{
							{
								Key:   []byte("proposer"),
								Value: []byte("FCAA001"),
								Index: true,
							},
						},
					},
				},
			},
			ResultEndBlock: abci.ResponseEndBlock{
				Events: []abci.Event{
					{
						Type: "end_event",
						Attributes: []abci.EventAttribute{
							{
								Key:   []byte("foo"),
								Value: []byte(fmt.Sprintf("%d", i)),
								Index: index,
							},
						},
					},
				},
			},
		}))
	}

	testCases := map[string]struct {
		q       *query.Query
		results []int64
	}{
		"block.height < 5": {
			q:       query.MustParse("block.height < 2"),
			results: []int64{1},
		},
		"block.height = 100": {
			q:       query.MustParse("block.height = 100"),
			results: []int64{},
		},
		"block.height = 5": {
			q:       query.MustParse("block.height = 5"),
			results: []int64{5},
		},
		"begin_event.key1 = 'value1'": {
			q:       query.MustParse("begin_event.key1 = 'value1'"),
			results: []int64{},
		},
		"begin_event.proposer = 'FCAA001'": {
			q:       query.MustParse("begin_event.proposer = 'FCAA001'"),
			results: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		"end_event.foo <= 5": {
			q:       query.MustParse("end_event.foo <= 5"),
			results: []int64{2, 4},
		},
		"end_event.foo >= 100": {
			q:       query.MustParse("end_event.foo >= 100"),
			results: []int64{1},
		},
		"block.height > 2 AND end_event.foo <= 8": {
			q:       query.MustParse("block.height > 2 AND end_event.foo <= 8"),
			results: []int64{4, 6, 8},
		},
		"begin_event.proposer CONTAINS 'FFFFFFF'": {
			q:       query.MustParse("begin_event.proposer CONTAINS 'FFFFFFF'"),
			results: []int64{},
		},
		"begin_event.proposer CONTAINS 'FCAA001'": {
			q:       query.MustParse("begin_event.proposer CONTAINS 'FCAA001'"),
			results: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			results, err := indexer.Search(context.Background(), tc.q)
			require.NoError(t, err)
			require.Equal(t, tc.results, results)
		})
	}
}

func TestBlockIndexerPruning(t *testing.T) {

	// init the block indexer
	prefixStore := store.NewPrefixKV(store.NewDefaultInMemoryKVStore(), []byte("block_events"))
	indexer := blockidxkv.New(prefixStore)
	numBlocks := uint64(100)

	numEvents := uint64(0)
	// index block data
	for i := uint64(1); i <= numBlocks; i++ {
		beginBlock := getBeginBlock()
		endBlock := getEndBlock()
		numEvents += uint64(len(beginBlock.Events))
		numEvents += uint64(len(endBlock.Events))
		indexer.Index(types.EventDataNewBlockHeader{
			Header:           types.Header{Height: int64(i)},
			ResultBeginBlock: beginBlock,
			ResultEndBlock:   endBlock,
		})
	}

	// query all blocks and receive events for all block heights
	queryString := fmt.Sprintf("block.height <= %d", numBlocks)
	q := query.MustParse(queryString)
	results, err := indexer.Search(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, numBlocks, uint64(len(results)))

	// prune indexer for all heights
	pruned, err := indexer.Prune(1, numBlocks+1, log.NewNopLogger())
	require.NoError(t, err)
	require.Equal(t, numBlocks+numEvents, pruned)

	// check the query returns empty
	results, err = indexer.Search(context.Background(), q)
	require.NoError(t, err)
	require.Equal(t, 0, len(results))

}

func getBeginBlock() abci.ResponseBeginBlock {
	if rand.Intn(2) == 1 {
		return abci.ResponseBeginBlock{
			Events: []abci.Event{
				{
					Type: "begin_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("proposer"),
							Value: []byte("FCAA001"),
							Index: true,
						},
					},
				},
			},
		}
	} else {
		return abci.ResponseBeginBlock{}
	}
}

func getEndBlock() abci.ResponseEndBlock {
	if rand.Intn(2) == 1 {
		return abci.ResponseEndBlock{
			Events: []abci.Event{
				{
					Type: "end_event",
					Attributes: []abci.EventAttribute{
						{
							Key:   []byte("foo"),
							Value: []byte("value"),
							Index: true,
						},
					},
				},
			},
		}
	} else {
		return abci.ResponseEndBlock{}
	}
}
