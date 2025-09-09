package dymension

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TEENonce represents the nonce data that gets signed by the TEE
// This is a temporary local definition until proto files are updated
type TEENonce struct {
	RollappId          string `protobuf:"bytes,2,opt,name=rollapp_id,json=rollappId,proto3" json:"rollapp_id,omitempty"`
	CurrHeight         uint64 `protobuf:"varint,1,opt,name=curr_height,json=currHeight,proto3" json:"curr_height,omitempty"`
	CurrStateRoot      []byte `protobuf:"bytes,3,opt,name=curr_state_root,json=currStateRoot,proto3" json:"curr_state_root,omitempty"`
	FinalizedHeight    uint64 `protobuf:"varint,4,opt,name=finalized_height,json=finalizedHeight,proto3" json:"finalized_height,omitempty"`
	FinalizedStateRoot []byte `protobuf:"bytes,5,opt,name=finalized_state_root,json=finalizedStateRoot,proto3" json:"finalized_state_root,omitempty"`
}

// MsgFastFinalizeWithTEE fast-finalizes rollapp state using TEE attestation
// This is a temporary local definition until proto files are updated
type MsgFastFinalizeWithTEE struct {
	Creator             string   `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator,omitempty"`
	AttestationToken    string   `protobuf:"bytes,4,opt,name=attestation_token,json=attestationToken,proto3" json:"attestation_token,omitempty"`
	Nonce               TEENonce `protobuf:"bytes,5,opt,name=nonce,proto3" json:"nonce"`
	CurrStateIndex      uint64   `protobuf:"varint,6,opt,name=curr_state_index,json=currStateIndex,proto3" json:"curr_state_index,omitempty"`
	FinalizedStateIndex uint64   `protobuf:"varint,7,opt,name=finalized_state_index,json=finalizedStateIndex,proto3" json:"finalized_state_index,omitempty"`
}

// Implement basic methods to satisfy protobuf interface
func (m *MsgFastFinalizeWithTEE) Reset()         { *m = MsgFastFinalizeWithTEE{} }
func (m *MsgFastFinalizeWithTEE) String() string { return "" }
func (*MsgFastFinalizeWithTEE) ProtoMessage()    {}

// ValidateBasic performs basic validation of the message
func (m MsgFastFinalizeWithTEE) ValidateBasic() error {
	if m.Creator == "" {
		return fmt.Errorf("invalid creator address")
	}
	if m.AttestationToken == "" {
		return fmt.Errorf("attestation token is required")
	}
	return nil
}

// GetSigners returns the expected signers for the message
func (m MsgFastFinalizeWithTEE) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{creator}
}

// Route returns the message router key
func (m MsgFastFinalizeWithTEE) Route() string {
	return "rollapp"
}

// Type returns the message type
func (m MsgFastFinalizeWithTEE) Type() string {
	return "FastFinalizeWithTEE"
}