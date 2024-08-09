package p2p

import (
	"github.com/dymensionxyz/dymint/p2p/pb"
	"github.com/dymensionxyz/dymint/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

/* -------------------------------------------------------------------------- */
/*                                 Event Data                                 */
/* -------------------------------------------------------------------------- */

// BlockData defines the struct of the data for each block sent via P2P
type BlockData struct {
	// Block is the block that was gossiped
	Block types.Block
	// Commit is the commit that was gossiped
	Commit types.Commit
}

// MarshalBinary encodes BlockData into binary form and returns it.
func (b *BlockData) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

// UnmarshalBinary decodes binary form of p2p received block into object.
func (b *BlockData) UnmarshalBinary(data []byte) error {
	var pbBlock pb.BlockData
	err := pbBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pbBlock)
	return err
}

// ToProto converts Data into protobuf representation and returns it.
func (b *BlockData) ToProto() *pb.BlockData {
	return &pb.BlockData{
		Block:  b.Block.ToProto(),
		Commit: b.Commit.ToProto(),
	}
}

// FromProto fills BlockData with data from its protobuf representation.
func (b *BlockData) FromProto(other *pb.BlockData) error {
	if err := b.Block.FromProto(other.Block); err != nil {
		return err
	}
	if err := b.Commit.FromProto(other.Commit); err != nil {
		return err
	}
	return nil
}

// Validate run basic validation on the p2p block received
func (b *BlockData) Validate(proposer *types.Sequencer) error {
	if err := b.Block.ValidateBasic(); err != nil {
		return err
	}
	if err := b.Commit.ValidateBasic(); err != nil {
		return err
	}
	if err := b.Commit.ValidateWithHeader(proposer, &b.Block.Header); err != nil {
		return err
	}
	abciData := tmtypes.Data{
		Txs: types.ToABCIBlockDataTxs(&b.Block.Data),
	}
	if b.Block.Header.DataHash != [32]byte(abciData.Hash()) {
		return types.ErrInvalidHeaderDataHash
	}
	return nil
}
