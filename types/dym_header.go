package types

import (
	"fmt"

	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	proto "github.com/gogo/protobuf/types"
	"github.com/tendermint/tendermint/crypto/merkle"
	cmtbytes "github.com/tendermint/tendermint/libs/bytes"
	_ "github.com/tendermint/tendermint/types"
)

// Additional data that the sequencer must sign over
type DymHeader struct {
	// must be the hash of the (merkle root) of the consensus messages
	ConsensusMessagesHash [32]byte
}

func NewDymHeader(cons []*proto.Any) *DymHeader {
	return &DymHeader{
		ConsensusMessagesHash: consMessagesHash(cons),
	}
}

func (d *DymHeader) ValidateBasic() error {
	if d == nil {
		return fmt.Errorf("dym header is nil")
	}
	return nil
}

func (d *DymHeader) Hash() cmtbytes.HexBytes {
	// 32 bytes long
	return merkle.HashFromByteSlices([][]byte{
		d.ConsensusMessagesHash[:],
		// can be extended with other things if we need to later
	})
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

func consMessagesHash(msgs []*proto.Any) [32]byte {
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
