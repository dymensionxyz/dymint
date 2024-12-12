package null

import (
	"context"
	"errors"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/dymensionxyz/dymint/indexers/txindex"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
)

var _ txindex.TxIndexer = (*TxIndex)(nil)

type TxIndex struct{}

func (txi *TxIndex) Get(hash []byte) (*abci.TxResult, error) {
	return nil, errors.New(`indexing is disabled (set 'tx_index = "kv"' in config)`)
}

func (txi *TxIndex) AddBatch(batch *txindex.Batch) error {
	return nil
}

func (txi *TxIndex) Index(result *abci.TxResult) error {
	return nil
}

func (txi *TxIndex) Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error) {
	return []*abci.TxResult{}, nil
}

func (txi *TxIndex) Prune(from, to uint64, logger log.Logger) (uint64, error) {
	return 0, nil
}
