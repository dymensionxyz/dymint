package types

import (
	"fmt"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	proto "github.com/gogo/protobuf/types"
	"github.com/tendermint/tendermint/crypto/merkle"
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	_ "github.com/tendermint/tendermint/types"
)

// a convenience struct to make computing easier
// persisted and over the wire types use a flat representation
type DymHeader struct {
	ConsensusMessagesHash [32]byte
}

func MakeDymHeader(consMessages []*proto.Any) DymHeader {
	return DymHeader{
		ConsensusMessagesHash: ConsMessagesHash(consMessages),
	}
}

func (h DymHeader) Hash() cmtbytes.HexBytes {
	// 32 bytes long
	return merkle.HashFromByteSlices([][]byte{
		h.ConsensusMessagesHash[:],
		// can be extended with other things if we need to later
	})
}

func (h *Header) SetDymHeader(dh DymHeader) {
	h.ConsensusMessagesHash = dh.ConsensusMessagesHash
}

func (h *Header) DymHash() cmtbytes.HexBytes {
	return DymHeader{
		ConsensusMessagesHash: h.ConsensusMessagesHash,
	}.Hash()
}

func dymHashFr(blocks []*Block) cmtbytes.HexBytes {
	// 32 bytes long
	bzz := make([][]byte, len(blocks))
	for i, block := range blocks {
		bzz[i] = block.Header.DymHash()
	}
	return merkle.HashFromByteSlices(bzz)
}

func (d *DymHeader) ToProto() *pb.DymHeader {
	if d == nil {
		// responsibility of caller to validate
		return nil
	}
	return &pb.DymHeader{
		ConsensusMessagesHash: d.ConsensusMessagesHash[:],
	}
}

func (d *DymHeader) FromProto(o *pb.DymHeader) error {
	if o == nil || len(o.ConsensusMessagesHash) != 32 {
		// the proto is invalid, or comes from an old version
		// we don't error here, because it's the responsibility of the validation func
		return nil
	}
	copy(d.ConsensusMessagesHash[:], o.ConsensusMessagesHash)
	return nil // just return error anyway to stick with pattern
}

func ConsMessagesHash(msgs []*proto.Any) [32]byte {
	bzz := make([][]byte, len(msgs))
	for i, msg := range msgs {
		var err error
		bzz[i], err = msg.Marshal()
		if err != nil {
			// Not obvious how to recover here. Shouldn't happen.
			panic(fmt.Errorf("marshal consensus message: %w", err))
		}
	}
	merkleRoot := merkle.HashFromByteSlices(bzz)
	ret := [32]byte{}
	copy(ret[:], merkleRoot) // merkleRoot is already 32 bytes
	return ret
}
