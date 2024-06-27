package interchain_da

import sdk "github.com/cosmos/cosmos-sdk/types"

func (m MsgSubmitBlob) ValidateBasic() error {
	// Validation is done on the DA layer side
	return nil
}

func (m MsgSubmitBlob) GetSigners() []sdk.AccAddress {
	signer, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{signer}
}
