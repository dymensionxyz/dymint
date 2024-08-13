package p2p

import (
	"github.com/dymensionxyz/dymint/p2p/pb"
	"github.com/dymensionxyz/dymint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

/* -------------------------------------------------------------------------- */
/*                                 Event Data                                 */
/* -------------------------------------------------------------------------- */

// P2PBlockEvent defines the struct of the event data for the Block sent via P2P
type P2PBlockEvent struct {
	// Block is the block that was gossiped
	Block types.Block
	// Commit is the commit that was gossiped
	Commit types.Commit
}

// MarshalBinary encodes GossipedBlock into binary form and returns it.
func (e *P2PBlockEvent) MarshalBinary() ([]byte, error) {
	return e.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of GossipedBlock into object.
func (e *P2PBlockEvent) UnmarshalBinary(data []byte) error {
	var pbGossipedBlock pb.GossipedBlock
	err := pbGossipedBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = e.FromProto(&pbGossipedBlock)
	return err
}

// ToProto converts Data into protobuf representation and returns it.
func (e *P2PBlockEvent) ToProto() *pb.GossipedBlock {
	return &pb.GossipedBlock{
		Block:  e.Block.ToProto(),
		Commit: e.Commit.ToProto(),
	}
}

// FromProto fills P2PBlock with data from its protobuf representation.
func (e *P2PBlockEvent) FromProto(other *pb.GossipedBlock) error {
	if err := e.Block.FromProto(other.Block); err != nil {
		return err
	}
	if err := e.Commit.FromProto(other.Commit); err != nil {
		return err
	}
	return nil
}

// Validate run basic validation on the p2p block
func (e *P2PBlockEvent) Validate(proposer *types.Sequencer) error {
	if err := e.Block.ValidateBasic(); err != nil {
		return err
	}
	if err := e.Commit.ValidateBasic(); err != nil {
		return err
	}
	if err := e.Commit.ValidateWithHeader(proposer, &e.Block.Header); err != nil {
		return err
	}
	abciData := tmtypes.Data{
		Txs: types.ToABCIBlockDataTxs(&e.Block.Data),
	}
	if e.Block.Header.DataHash != [32]byte(abciData.Hash()) {
		return types.ErrInvalidHeaderDataHash
	}
	return nil
}
