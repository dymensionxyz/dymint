package null

import (
	"context"
	"errors"

	"github.com/tendermint/tendermint/libs/log"

	indexer "github.com/dymensionxyz/dymint/indexers/blockindexer"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	"github.com/tendermint/tendermint/types"
)

var _ indexer.BlockIndexer = (*BlockerIndexer)(nil)

// TxIndex implements a no-op block indexer.
type BlockerIndexer struct{}

func (idx *BlockerIndexer) Has(height int64) (bool, error) {
	return false, errors.New(`indexing is disabled (set 'tx_index = "kv"' in config)`)
}

func (idx *BlockerIndexer) Index(types.EventDataNewBlockHeader) error {
	return nil
}

func (idx *BlockerIndexer) Search(ctx context.Context, q *query.Query) ([]int64, error) {
	return []int64{}, nil
}

func (idx *BlockerIndexer) Prune(from, to uint64, logger log.Logger) (uint64, error) {
	return 0, nil
}
