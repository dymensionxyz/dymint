package rollapp

import (
	"math"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgUpdateState = "update_state"

var _ sdk.Msg = &MsgUpdateState{}

func (msg *MsgUpdateState) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgUpdateState) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return errorsmod.Wrapf(ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	
	if msg.NumBlocks == uint64(0) {
		return errorsmod.Wrap(ErrInvalidNumBlocks, "number of blocks can not be zero")
	}

	if msg.NumBlocks > math.MaxUint64-msg.StartHeight {
		return errorsmod.Wrapf(ErrInvalidNumBlocks, "numBlocks(%d) + startHeight(%d) exceeds max uint64", msg.NumBlocks, msg.StartHeight)
	}

	
	if uint64(len(msg.BDs.BD)) != msg.NumBlocks {
		return errorsmod.Wrapf(ErrInvalidNumBlocks, "number of blocks (%d) != number of block descriptors(%d)", msg.NumBlocks, len(msg.BDs.BD))
	}

	
	if msg.StartHeight == 0 {
		return errorsmod.Wrapf(ErrWrongBlockHeight, "StartHeight must be greater than zero")
	}

	
	for bdIndex := uint64(0); bdIndex < msg.NumBlocks; bdIndex += 1 {
		if msg.BDs.BD[bdIndex].Height != msg.StartHeight+bdIndex {
			return ErrInvalidBlockSequence
		}
		
		if len(msg.BDs.BD[bdIndex].StateRoot) != 32 {
			return errorsmod.Wrapf(ErrInvalidStateRoot, "StateRoot of block high (%d) must be 32 byte array. But received (%d) bytes",
				msg.BDs.BD[bdIndex].Height, len(msg.BDs.BD[bdIndex].StateRoot))
		}
	}

	return nil
}
