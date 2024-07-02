package interchain_da

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (m *MsgSubmitBlob) ValidateBasic() error {
	// Tha validation occurs on the client side.
	return nil
}

func (m *MsgSubmitBlob) GetSigners() []sdk.AccAddress {
	signer, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{signer}
}

type BlobID uint64

// Module name and store keys.
const (
	// ModuleName defines the module name
	ModuleName = "interchain_da"

	ModuleNameCLI = "interchain-da"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

const (
	ParamsByte uint8 = iota
	BlobIDByte
	BlobMetadataByte
	PruningHeightByte
)

func ParamsPrefix() []byte {
	return []byte{ParamsByte}
}

func BlobIDPrefix() []byte {
	return []byte{BlobIDByte}
}

func BlobMetadataPrefix() []byte {
	return []byte{BlobMetadataByte}
}

func PruningHeightPrefix() []byte {
	return []byte{PruningHeightByte}
}
