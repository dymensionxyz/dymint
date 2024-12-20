package types

import (
	"encoding"
	"time"

	proto "github.com/gogo/protobuf/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// Header defines the structure of Dymint block header.
type Header struct {
	// Block and App version
	Version Version

	Height uint64
	Time   int64 // UNIX time in nanoseconds. Use int64 as Golang stores UNIX nanoseconds in int64.

	// prev block info
	LastHeaderHash [32]byte

	// hashes of block data
	LastCommitHash [32]byte // commit from sequencer(s) from the last block
	DataHash       [32]byte // Block.Data root aka Transactions
	ConsensusHash  [32]byte // consensus params for current block
	AppHash        [32]byte // state after applying txs from height-1

	// Root hash of all results from the txs from the previous block.
	// This is ABCI specific but smart-contract chains require some way of committing
	// to transaction receipts/results.
	LastResultsHash [32]byte

	// Note that the address can be derived from the pubkey which can be derived
	// from the signature when using secp256k.
	// We keep this in case users choose another signature format where the
	// pubkey can't be recovered by the signature (e.g. ed25519).
	ProposerAddress []byte // original proposer of the block

	// Hash of proposer validatorSet (compatible with tendermint)
	SequencerHash [32]byte
	// Hash of the next proposer validatorSet (compatible with tendermint)
	NextSequencersHash [32]byte

	// The Chain ID
	ChainID string

	// The following fields are added on top of the normal TM header, for dymension purposes
	// Note: LOSSY when converted to tendermint (squashed into a single hash)
	ConsensusMessagesHash [32]byte // must be the hash of the (merkle root) of the consensus messages
}

func (h Header) GetTimestamp() time.Time {
	return time.Unix(0, h.Time)
}

var (
	_ encoding.BinaryMarshaler   = &Header{}
	_ encoding.BinaryUnmarshaler = &Header{}
)

// Version captures the consensus rules for processing a block in the blockchain,
// including all blockchain data structures and the rules of the application's
// state transition machine.
// This is equivalent to the tmversion.Consensus type in Tendermint.
type Version struct {
	Block uint64
	App   uint64
}

// Block defines the structure of Dymint block.
type Block struct {
	Header     Header
	Data       Data
	LastCommit Commit
}

func (b Block) SizeBytes() int {
	return b.ToProto().Size()
}

func (b *Block) GetRevision() uint64 {
	return b.Header.Version.App
}

var (
	_ encoding.BinaryMarshaler   = &Block{}
	_ encoding.BinaryUnmarshaler = &Block{}
)

// Data defines Dymint block data.
type Data struct {
	Txs                    Txs
	IntermediateStateRoots IntermediateStateRoots
	Evidence               EvidenceData
	ConsensusMessages      []*proto.Any
}

// EvidenceData defines how evidence is stored in block.
type EvidenceData struct {
	Evidence []Evidence
}

// Commit contains evidence of block creation.
type Commit struct {
	Height     uint64
	HeaderHash [32]byte
	// TODO(omritoptix): Change from []Signature to Signature as it should be one signature per block
	Signatures  []Signature
	TMSignature tmtypes.CommitSig
}

func (c Commit) SizeBytes() int {
	return c.ToProto().Size()
}

// Signature represents signature of block creator.
type Signature []byte

// IntermediateStateRoots describes the state between transactions.
// They are required for fraud proofs.
type IntermediateStateRoots struct {
	RawRootsList [][]byte
}

func GetLastCommitHash(lastCommit *Commit, header *Header) []byte {
	lastABCICommit := ToABCICommit(lastCommit, header)
	return lastABCICommit.Hash()
}

// GetDataHash returns the hash of the block data to be set in the block header.
// Doesn't include consensus messages because we want to avoid touching
// fundamental primitive and allow tx inclusion proofs.
func GetDataHash(block *Block) []byte {
	abciData := tmtypes.Data{
		Txs: ToABCIBlockDataTxs(&block.Data),
	}
	return abciData.Hash()
}
