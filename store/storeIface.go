package store

import (
	"github.com/dymensionxyz/dymint/types"
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmtypes "github.com/tendermint/tendermint/types"
)

// KV encapsulates key-value store abstraction, in minimalistic interface.
//
// KV MUST be thread safe.
type KV interface {
	Get(key []byte) ([]byte, error)          // Get gets the value for a key.
	Set(key []byte, value []byte) error      // Set updates the value for a key.
	Delete(key []byte) error                 // Delete deletes a key.
	NewBatch() KVBatch                       // NewBatch creates a new batch.
	PrefixIterator(prefix []byte) KVIterator // PrefixIterator creates iterator to traverse given prefix.
	Close() error                            // Close closes the store.
}

// KVBatch enables batching of transactions.
type KVBatch interface {
	Set(key, value []byte) error // Accumulates KV entries in a transaction.
	Delete(key []byte) error     // Deletes the given key.
	Commit() error               // Commits the transaction.
	Discard()                    // Discards the transaction.
}

// KVIterator enables traversal over a given prefix.
type KVIterator interface {
	Valid() bool
	Next()
	Key() []byte
	Value() []byte
	Error() error
	Discard()
}

// Store is minimal interface for storing and retrieving blocks, commits and state.
type Store interface {
	// NewBatch creates a new db batch.
	NewBatch() KVBatch

	// SaveBlock saves block along with its seen commit (which will be included in the next block).
	SaveBlock(block *types.Block, commit *types.Commit, batch KVBatch) (KVBatch, error)

	// LoadBlock returns block at given height, or error if it's not found in Store.
	LoadBlock(height uint64) (*types.Block, error)
	// LoadBlockByHash returns block with given block header hash, or error if it's not found in Store.
	LoadBlockByHash(hash [32]byte) (*types.Block, error)

	// SaveBlockResponses saves block responses (events, tx responses, validator set updates, etc) in Store.
	SaveBlockResponses(height uint64, responses *tmstate.ABCIResponses, batch KVBatch) (KVBatch, error)

	// LoadBlockResponses returns block results at given height, or error if it's not found in Store.
	LoadBlockResponses(height uint64) (*tmstate.ABCIResponses, error)

	// LoadCommit returns commit for a block at given height, or error if it's not found in Store.
	LoadCommit(height uint64) (*types.Commit, error)
	// LoadCommitByHash returns commit for a block with given block header hash, or error if it's not found in Store.
	LoadCommitByHash(hash [32]byte) (*types.Commit, error)

	// SaveState updates state saved in Store. Only one State is stored.
	// If there is no State in Store, state will be saved.
	SaveState(state *types.State, batch KVBatch) (KVBatch, error)

	// LoadState returns last state saved with UpdateState.
	LoadState() (*types.State, error)

	SaveValidators(height uint64, validatorSet *tmtypes.ValidatorSet, batch KVBatch) (KVBatch, error)

	LoadValidators(height uint64) (*tmtypes.ValidatorSet, error)

	PruneBlocks(from, to uint64) (uint64, error)

	Close() error
}
