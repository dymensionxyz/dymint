package proto

import (
	cosmos "github.com/cosmos/cosmos-sdk/codec/types"
	gogo "github.com/gogo/protobuf/types"
)

func GogoToCosmos(v *gogo.Any) *cosmos.Any {
	if v == nil {
		return nil
	}
	return &cosmos.Any{
		TypeUrl: v.TypeUrl,
		Value:   v.Value,
	}
}

func CosmosToGogo(v *cosmos.Any) *gogo.Any {
	if v == nil {
		return nil
	}
	return &gogo.Any{
		TypeUrl: v.TypeUrl,
		Value:   v.Value,
	}
}
