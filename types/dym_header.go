package types

import (
	"fmt"

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
	copy(h.ConsensusMessagesHash[:], dh.ConsensusMessagesHash[:])
}

func (h *Header) DymHash() cmtbytes.HexBytes {
	ret := DymHeader{}
	copy(ret.ConsensusMessagesHash[:], h.ConsensusMessagesHash[:])
	return ret.Hash()
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
