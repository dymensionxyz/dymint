package p2p

import (
	"github.com/dymensionxyz/dymint/p2p/pb"
	"github.com/dymensionxyz/dymint/types"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

type BlockData struct {
	Block types.Block

	Commit types.Commit
}

func (b *BlockData) MarshalBinary() ([]byte, error) {
	return b.ToProto().Marshal()
}

func (b *BlockData) UnmarshalBinary(data []byte) error {
	var pbBlock pb.BlockData
	err := pbBlock.Unmarshal(data)
	if err != nil {
		return err
	}
	err = b.FromProto(&pbBlock)
	return err
}

func (b *BlockData) ToProto() *pb.BlockData {
	return &pb.BlockData{
		Block:  b.Block.ToProto(),
		Commit: b.Commit.ToProto(),
	}
}

func (b *BlockData) FromProto(other *pb.BlockData) error {
	if err := b.Block.FromProto(other.Block); err != nil {
		return err
	}
	if err := b.Commit.FromProto(other.Commit); err != nil {
		return err
	}
	return nil
}

func (b *BlockData) Validate(proposerPubKey tmcrypto.PubKey) error {
	if err := b.Block.ValidateBasic(); err != nil {
		return err
	}

	if err := b.Commit.ValidateWithHeader(proposerPubKey, &b.Block.Header); err != nil {
		return err
	}
	return nil
}
