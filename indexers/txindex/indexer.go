package txindex

import (
	"context"
	"errors"

	"github.com/tendermint/tendermint/libs/log"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
)

type TxIndexer interface {
	AddBatch(b *Batch) error

	Index(result *abci.TxResult) error

	Get(hash []byte) (*abci.TxResult, error)

	Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error)

	Prune(from, to uint64, logger log.Logger) (uint64, error)
}

type Batch struct {
	Height int64
	Ops    []*abci.TxResult
}

func NewBatch(n int64, height int64) *Batch {
	return &Batch{
		Height: height,
		Ops:    make([]*abci.TxResult, n),
	}
}

func (b *Batch) Add(result *abci.TxResult) error {
	b.Ops[result.Index] = result
	return nil
}

func (b *Batch) Size() int {
	return len(b.Ops)
}

var ErrorEmptyHash = errors.New("transaction hash cannot be empty")
