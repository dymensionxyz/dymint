package types

import (
	"bytes"
	"fmt"

	proto "github.com/gogo/protobuf/types"
	"github.com/tendermint/tendermint/crypto/merkle"
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
)

// legacy version of dymint which did not support consensus message hashing
const blockVersionWithDefaultEvidenceHash = 0

var defaultEvidenceHash = new(tmtypes.EvidenceData).Hash()

// we overload tendermint header evidence hash with our own stuff
// (we don't need evidence, because we don't use comet)
func evidenceHash(header *Header) cmtbytes.HexBytes {
	if header.Version.Block == blockVersionWithDefaultEvidenceHash {
		return defaultEvidenceHash
	}
	return header.Extra.hash()
}

// header corresponds to block?
//
// valid possibilities:
// - old block version + extra data not populated + hash is default
// - new block version + extra data populated + hash is something // TODO: empty?
func validateExtra(block *Block, header *Header) error {
	zero := ExtraSignedData{}
	if bytes.Equal(evidenceHash(header), defaultEvidenceHash) {
		if header.Extra != zero {
			return fmt.Errorf("unexpected extra data in block with default evidence hash")
		}
		return nil

	}
	zero := ExtraSignedData{}
	if block.Header.Extra == zero && block.Data.ConsensusMessages {

	}
}

func (e ExtraSignedData) validateBlock(block *Block) error {
	expect := e
	got, err := ExtraSignedData{}.fromBlock(block)
	if err != nil {
		return fmt.Errorf("from block: %w", err)
	}
	expectH := expect.hash()
	gotH := got.hash()
	if !bytes.Equal(expectH, gotH) {
		return fmt.Errorf("hash mismatch: expected %X, got %X", expectH, gotH)
	}
	return nil
}

func (e ExtraSignedData) hash() []byte {
	return merkle.HashFromByteSlices([][]byte{e.ConsensusMessagesHash[:]})
}

func (e ExtraSignedData) fromBlock(block *Block) (ExtraSignedData, error) {
	ret := ExtraSignedData{}
	var err error
	ret.ConsensusMessagesHash, err = consMessagesHash(block.Data.ConsensusMessages)
	if err != nil {
		return ExtraSignedData{}, fmt.Errorf("consensus messages hash: %w", err)
	}
	return ret, nil
}

func consMessagesHash(msgs []*proto.Any) ([32]byte, error) {
	bzz := make([][]byte, len(msgs))
	for i, msg := range msgs {
		var err error
		bzz[i], err = msg.Marshal()
		if err != nil {
			return [32]byte{}, fmt.Errorf("marshal consensus message: %w", err)
		}
	}
	merkleRoot := merkle.HashFromByteSlices(bzz)
	ret := [32]byte{}
	copy(ret[:], merkleRoot) // merkleRoot is already 32 bytes
	return ret, nil
}
