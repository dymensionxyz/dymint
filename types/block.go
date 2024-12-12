package types

import (
	"encoding"
	"time"

	proto "github.com/gogo/protobuf/types"
	tmtypes "github.com/tendermint/tendermint/types"
)


type Header struct {
	
	Version Version

	Height uint64
	Time   int64 

	
	LastHeaderHash [32]byte

	
	LastCommitHash [32]byte 
	DataHash       [32]byte 
	ConsensusHash  [32]byte 
	AppHash        [32]byte 

	
	
	
	LastResultsHash [32]byte

	
	
	
	
	ProposerAddress []byte 

	
	SequencerHash [32]byte
	
	NextSequencersHash [32]byte

	
	ChainID string
}

func (h Header) GetTimestamp() time.Time {
	return time.Unix(0, h.Time)
}

var (
	_ encoding.BinaryMarshaler   = &Header{}
	_ encoding.BinaryUnmarshaler = &Header{}
)





type Version struct {
	Block uint64
	App   uint64
}


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


type Data struct {
	Txs                    Txs
	IntermediateStateRoots IntermediateStateRoots
	Evidence               EvidenceData
	ConsensusMessages      []*proto.Any
}


type EvidenceData struct {
	Evidence []Evidence
}


type Commit struct {
	Height     uint64
	HeaderHash [32]byte
	
	Signatures  []Signature
	TMSignature tmtypes.CommitSig
}

func (c Commit) SizeBytes() int {
	return c.ToProto().Size()
}


type Signature []byte



type IntermediateStateRoots struct {
	RawRootsList [][]byte
}

func GetLastCommitHash(lastCommit *Commit, header *Header) []byte {
	lastABCICommit := ToABCICommit(lastCommit, header)
	return lastABCICommit.Hash()
}


func GetDataHash(block *Block) []byte {
	abciData := tmtypes.Data{
		Txs: ToABCIBlockDataTxs(&block.Data),
	}
	return abciData.Hash()
}
