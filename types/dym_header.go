package types

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
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

func (d *DymHeader) Hash() cmtbytes.HexBytes {
	// 32 bytes long
	return merkle.HashFromByteSlices([][]byte{
		d.ConsensusMessagesHash[:],
		// can be extended with other things if we need to later
	})
}

func (d *DymHeader) ToProto() *dymint.DymHeader {
	if d == nil {
		return nil // for tests
	}
	return &dymint.DymHeader{
		ConsensusMessagesHash: d.ConsensusMessagesHash[:],
	}
}

func (d *DymHeader) FromProto(o *pb.DymHeader) error {
	if o == nil || len(o.ConsensusMessagesHash) != 32 {
		return fmt.Errorf("missing or corrupted - check software version is same as block: %w", ErrInvalidDymHeader)
	}
	copy(d.ConsensusMessagesHash[:], o.ConsensusMessagesHash)
	return nil
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
