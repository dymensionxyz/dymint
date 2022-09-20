package store

import (
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmtypes "github.com/tendermint/tendermint/types"

	"github.com/dymensionxyz/dymint/types"
)

// Store is minimal interface for storing and retrieving blocks, commits and state.
type Store interface {

	// NewBatch creates a new db batch.
	NewBatch() Batch

	// Height returns height of the highest block in store.
	Height() uint64

	// SetHeight sets the height saved in the Store if it is higher than the existing height.
	SetHeight(height uint64)

	// SaveBlock saves block along with its seen commit (which will be included in the next block).
	SaveBlock(block *types.Block, commit *types.Commit) error

	// SaveBlockWithBatch saves block along with its seen commit (which will be included in the next block).
	SaveBlockWithBatch(block *types.Block, commit *types.Commit, batch Batch) (Batch, error)

	// LoadBlock returns block at given height, or error if it's not found in Store.
	LoadBlock(height uint64) (*types.Block, error)
	// LoadBlockByHash returns block with given block header hash, or error if it's not found in Store.
	LoadBlockByHash(hash [32]byte) (*types.Block, error)

	// SaveBlockResponses saves block responses (events, tx responses, validator set updates, etc) in Store.
	SaveBlockResponses(height uint64, responses *tmstate.ABCIResponses) error

	// SaveBlockResponsesWithBatch saves block responses (events, tx responses, validator set updates, etc) in Store.
	SaveBlockResponsesWithBatch(height uint64, responses *tmstate.ABCIResponses, batch Batch) (Batch, error)

	// LoadBlockResponses returns block results at given height, or error if it's not found in Store.
	LoadBlockResponses(height uint64) (*tmstate.ABCIResponses, error)

	// LoadCommit returns commit for a block at given height, or error if it's not found in Store.
	LoadCommit(height uint64) (*types.Commit, error)
	// LoadCommitByHash returns commit for a block with given block header hash, or error if it's not found in Store.
	LoadCommitByHash(hash [32]byte) (*types.Commit, error)

	// UpdateState updates state saved in Store. Only one State is stored.
	// If there is no State in Store, state will be saved.
	UpdateState(state types.State) error

	// UpdateStateWithBatch updates state saved in Store. Only one State is stored.
	// If there is no State in Store, state will be saved.
	UpdateStateWithBatch(state types.State, batch Batch) (Batch, error)

	// LoadState returns last state saved with UpdateState.
	LoadState() (types.State, error)

	SaveValidators(height uint64, validatorSet *tmtypes.ValidatorSet) error

	SaveValidatorsWithBatch(height uint64, validatorSet *tmtypes.ValidatorSet, batch Batch) (Batch, error)

	LoadValidators(height uint64) (*tmtypes.ValidatorSet, error)
}
