package daclient

import (
	"encoding/json"
	"time"

	"github.com/celestiaorg/go-square/merkle"
	"github.com/tendermint/tendermint/crypto"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmversion "github.com/tendermint/tendermint/proto/tendermint/version"
	tmtypes "github.com/tendermint/tendermint/types"
)

// Header defines the structure of a Tendermint block header.
// NOTE: changes to the Header should be duplicated in:
// - header.Hash()
// - abci.Header
// - https://github.com/tendermint/tendermint/blob/v0.34.x/spec/blockchain/blockchain.md
type RawHeader struct {
	// basic block info
	Version tmversion.Consensus `json:"version"`
	ChainID string              `json:"chain_id"`
	Height  int64               `json:"height,string"`
	Time    time.Time           `json:"time"`

	// prev block info
	LastBlockID tmtypes.BlockID `json:"last_block_id"`

	// hashes of block data
	LastCommitHash tmbytes.HexBytes `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       tmbytes.HexBytes `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash     tmbytes.HexBytes `json:"validators_hash"`      // validators for the current block
	NextValidatorsHash tmbytes.HexBytes `json:"next_validators_hash"` // validators for the next block
	ConsensusHash      tmbytes.HexBytes `json:"consensus_hash"`       // consensus params for current block
	AppHash            tmbytes.HexBytes `json:"app_hash"`             // state after txs from the previous block
	// root hash of all results from the txs from the previous block
	// see `deterministicResponseDeliverTx` to understand which parts of a tx is hashed into here
	LastResultsHash tmbytes.HexBytes `json:"last_results_hash"`

	// consensus info
	EvidenceHash    tmbytes.HexBytes `json:"evidence_hash"`    // evidence included in the block
	ProposerAddress crypto.Address   `json:"proposer_address"` // original proposer of the block
}

// DataAvailabilityHeader (DAHeader) contains the row and column roots of the
// erasure coded version of the data in Block.Data. The original Block.Data is
// split into shares and arranged in a square of width squareSize. Then, this
// square is "extended" into an extended data square (EDS) of width 2*squareSize
// by applying Reed-Solomon encoding. For details see Section 5.2 of
// https://arxiv.org/abs/1809.09044 or the Celestia specification:
// https://github.com/celestiaorg/celestia-specs/blob/master/src/specs/data_structures.md#availabledataheader
type DataAvailabilityHeader struct {
	// RowRoot_j = root((M_{j,1} || M_{j,2} || ... || M_{j,2k} ))
	RowRoots [][]byte `json:"row_roots"`
	// ColumnRoot_j = root((M_{1,j} || M_{2,j} || ... || M_{2k,j} ))
	ColumnRoots [][]byte `json:"column_roots"`
	// hash is the Merkle root of the row and column roots. This field is the
	// memoized result from `Hash()`.
	hash []byte
}

// ExtendedHeader represents a wrapped "raw" header that includes
// information necessary for Celestia Nodes to be notified of new
// block headers and perform Data Availability Sampling.
type ExtendedHeader struct {
	RawHeader    `json:"header"`
	Commit       *Commit                 `json:"commit"`
	ValidatorSet *ValidatorSet           `json:"validator_set"`
	DAH          *DataAvailabilityHeader `json:"dah"`
}

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The signatures are in order of address to preserve the bonded
	// ValidatorSet order.
	// Any peer with a block can gossip signatures by index with a peer without
	// recalculating the active ValidatorSet.
	Height     int64       `json:"height"`
	Round      int32       `json:"round"`
	BlockID    BlockID     `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`
}

// CommitSig is a part of the Vote included in a Commit.
type CommitSig struct {
	BlockIDFlag      BlockIDFlag    `json:"block_id_flag"`
	ValidatorAddress crypto.Address `json:"validator_address"`
	Timestamp        time.Time      `json:"timestamp"`
	Signature        []byte         `json:"signature"`
}

// BlockIDFlag indicates which BlockID the signature is for.
type BlockIDFlag byte

// BlockID
type BlockID struct {
	Hash          tmbytes.HexBytes `json:"hash"`
	PartSetHeader PartSetHeader    `json:"parts"`
}

type PartSetHeader struct {
	Total uint32           `json:"total"`
	Hash  tmbytes.HexBytes `json:"hash"`
}

// ValidatorSet represent a set of *Validator at a given height.
//
// The validators can be fetched by address or index.
// The index is in order of .VotingPower, so the indices are fixed for all
// rounds of a given blockchain height - ie. the validators are sorted by their
// voting power (descending). Secondary index - .Address (ascending).
//
// On the other hand, the .ProposerPriority of each validator and the
// designated .GetProposer() of a set changes every round, upon calling
// .IncrementProposerPriority().
//
// NOTE: Not goroutine-safe.
// NOTE: All get/set to validators should copy the value for safety.
type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`
}

// Volatile state for each Validator
// NOTE: The ProposerPriority is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address     crypto.Address `json:"address"`
	PubKey      crypto.PubKey  `json:"pub_key"`
	VotingPower int64          `json:"voting_power"`

	ProposerPriority int64 `json:"proposer_priority"`
}

// Hash computes the Merkle root of the row and column roots. Hash memoizes the
// result in `DataAvailabilityHeader.hash`.
func (dah *DataAvailabilityHeader) Hash() []byte {
	if dah == nil {
		return merkle.HashFromByteSlices(nil)
	}
	if len(dah.hash) != 0 {
		return dah.hash
	}

	rowsCount := len(dah.RowRoots)
	slices := make([][]byte, rowsCount+rowsCount)
	copy(slices[0:rowsCount], dah.RowRoots)
	copy(slices[rowsCount:], dah.ColumnRoots)
	// The single data root is computed using a simple binary merkle tree.
	// Effectively being root(rowRoots || columnRoots):
	dah.hash = merkle.HashFromByteSlices(slices)
	return dah.hash
}

// MarshalJSON marshals an ExtendedHeader to JSON. The ValidatorSet is wrapped with amino encoding,
// to be able to unmarshal the crypto.PubKey type back from JSON.
func (eh *ExtendedHeader) MarshalJSON() ([]byte, error) {
	type Alias ExtendedHeader
	validatorSet, err := tmjson.Marshal(eh.ValidatorSet)
	if err != nil {
		return nil, err
	}
	rawHeader, err := tmjson.Marshal(eh.RawHeader)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&struct {
		RawHeader    json.RawMessage `json:"header"`
		ValidatorSet json.RawMessage `json:"validator_set"`
		*Alias
	}{
		ValidatorSet: validatorSet,
		RawHeader:    rawHeader,
		Alias:        (*Alias)(eh),
	})
}

// UnmarshalJSON unmarshals an ExtendedHeader from JSON. The ValidatorSet is wrapped with amino
// encoding, to be able to unmarshal the crypto.PubKey type back from JSON.
func (eh *ExtendedHeader) UnmarshalJSON(data []byte) error {
	type Alias ExtendedHeader
	aux := &struct {
		RawHeader    json.RawMessage `json:"header"`
		ValidatorSet json.RawMessage `json:"validator_set"`
		*Alias
	}{
		Alias: (*Alias)(eh),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	valSet := new(ValidatorSet)
	if err := tmjson.Unmarshal(aux.ValidatorSet, valSet); err != nil {
		return err
	}
	rawHeader := new(RawHeader)
	if err := tmjson.Unmarshal(aux.RawHeader, rawHeader); err != nil {
		return err
	}

	eh.ValidatorSet = valSet
	eh.RawHeader = *rawHeader
	return nil
}
