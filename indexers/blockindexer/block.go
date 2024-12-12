package indexer

import (
	"context"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/types"
)


type BlockIndexer interface {
	
	
	Has(height int64) (bool, error)

	
	Index(types.EventDataNewBlockHeader) error

	
	
	Search(ctx context.Context, q *query.Query) ([]int64, error)

	
	Prune(from, to uint64, logger log.Logger) (uint64, error)
}
