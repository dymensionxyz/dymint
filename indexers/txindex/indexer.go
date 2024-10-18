package txindex

import (
	"context"
	"errors"

	"github.com/tendermint/tendermint/libs/log"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
)

// TxIndexer interface defines methods to index and search transactions.
type TxIndexer interface {
	// AddBatch analyzes, indexes and stores a batch of transactions.
	AddBatch(b *Batch) error

	// Index analyzes, indexes and stores a single transaction.
	Index(result *abci.TxResult) error

	// Get returns the transaction specified by hash or nil if the transaction is not indexed
	// or stored.
	Get(hash []byte) (*abci.TxResult, error)

	// Search allows you to query for transactions.
	Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error)

	// Delete index entries for the heights between from (included) and to (not included). It returns heights pruned
	Prune(from, to uint64, logger log.Logger) (uint64, error)
}

// Batch groups together multiple Index operations to be performed at the same time.
// NOTE: Batch is NOT thread-safe and must not be modified after starting its execution.
type Batch struct {
	Height int64
	Ops    []*abci.TxResult
}

// NewBatch creates a new Batch.
func NewBatch(n int64, height int64) *Batch {
	return &Batch{
		Height: height,
		Ops:    make([]*abci.TxResult, n),
	}
}

// Add or update an entry for the given result.Index.
func (b *Batch) Add(result *abci.TxResult) error {
	b.Ops[result.Index] = result
	return nil
}

// Size returns the total number of operations inside the batch.
func (b *Batch) Size() int {
	return len(b.Ops)
}

// ErrorEmptyHash indicates empty hash
var ErrorEmptyHash = errors.New("transaction hash cannot be empty")
