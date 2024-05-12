package types

import (
	"fmt"
	"sync/atomic"
	"time"

	tmtypes "github.com/tendermint/tendermint/types"

	// TODO(tzdybal): copy to local project?
	tmstate "github.com/tendermint/tendermint/proto/tendermint/state"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	"github.com/tendermint/tendermint/types"
	"github.com/tendermint/tendermint/version"
)

// State contains information about current state of the blockchain.
type State struct {
	Version tmstate.Version

	// immutable
	ChainID       string
	InitialHeight uint64 // should be 1, not 0, when starting from height 1

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight uint64
	LastBlockID     types.BlockID
	LastBlockTime   time.Time

	// In the MVP implementation, there will be only one Validator
	NextValidators              *types.ValidatorSet
	Validators                  *types.ValidatorSet
	LastValidators              *types.ValidatorSet
	LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	ConsensusParams                  tmproto.ConsensusParams
	LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash [32]byte

	// LastStore height is the last height we've saved to the store.
	LastStoreHeight uint64

	// BaseHeight is the height of the first block we have in store after pruning.
	BaseHeight uint64

	// the latest AppHash we've received from calling abci.Commit()
	AppHash [32]byte
}

// FIXME: move from types package
// NewFromGenesisDoc reads blockchain State from genesis.
func NewFromGenesisDoc(genDoc *types.GenesisDoc) (State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return State{}, fmt.Errorf("in genesis doc: %w", err)
	}

	var validatorSet, nextValidatorSet *types.ValidatorSet
	validatorSet = types.NewValidatorSet(nil)
	nextValidatorSet = types.NewValidatorSet(nil)

	// InitStateVersion sets the Consensus.Block and Software versions,
	// but leaves the Consensus.App version blank.
	// The Consensus.App version will be set during the Handshake, once
	// we hear from the app what protocol version it is running.
	var InitStateVersion = tmstate.Version{
		Consensus: tmversion.Consensus{
			Block: version.BlockProtocol,
			App:   0,
		},
		Software: version.TMCoreSemVer,
	}

	s := State{
		Version:       InitStateVersion,
		ChainID:       genDoc.ChainID,
		InitialHeight: uint64(genDoc.InitialHeight),

		LastBlockHeight: 0,
		LastBlockID:     types.BlockID{},
		LastBlockTime:   genDoc.GenesisTime,

		NextValidators:              nextValidatorSet,
		Validators:                  validatorSet,
		LastValidators:              types.NewValidatorSet(nil),
		LastHeightValidatorsChanged: genDoc.InitialHeight,

		ConsensusParams:                  *genDoc.ConsensusParams,
		LastHeightConsensusParamsChanged: genDoc.InitialHeight,

		BaseHeight: 0,
	}
	copy(s.AppHash[:], genDoc.AppHash)

	return s, nil
}

func (s *State) IsGenesis() bool {
	return s.LastBlockHeight == 0
}

// SetHeight sets the height saved in the Store if it is higher than the existing height
// returns OK if the value was updated successfully or did not need to be updated
func (s *State) SetHeight(height uint64) bool {
	ok := true
	storeHeight := s.Height()
	if height > storeHeight {
		ok = atomic.CompareAndSwapUint64(&s.LastBlockHeight, storeHeight, height)
	}
	return ok
}

// Height returns height of the highest block saved in the Store.
func (s *State) Height() uint64 {
	return uint64(s.LastBlockHeight)
}

// NextHeight returns the next height that expected to be stored in store.
func (s *State) NextHeight() uint64 {
	return s.Height() + 1
}

// SetBase sets the base height if it is higher than the existing base height
// returns OK if the value was updated successfully or did not need to be updated
func (s *State) SetBase(height uint64) {
	s.BaseHeight = height
}

// Base returns height of the earliest block saved in the Store.
func (s *State) Base() uint64 {
	return s.BaseHeight
}

// SetABCICommitResult
func (s *State) SetABCICommitResult(resp *tmstate.ABCIResponses, appHash []byte, height uint64) {
	copy(s.AppHash[:], appHash[:])
	copy(s.LastResultsHash[:], tmtypes.NewResults(resp.DeliverTxs).Hash())

	s.LastValidators = s.Validators.Copy()
	s.LastStoreHeight = height
	s.SetHeight(height)
}
