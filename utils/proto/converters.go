package proto

import (
	"strings"

	cosmos "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
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

func FromProtoMsgToAny(msg proto.Message) *gogo.Any {
	theType, err := proto.Marshal(msg)
	if err != nil {
		return nil
	}

	typeUrl := strings.Replace(proto.MessageName(msg), "dymension_rdk", "rollapp", 1)

	return &gogo.Any{
		TypeUrl: typeUrl,
		Value:   theType,
	}
}

func FromProtoMsgSliceToAnySlice(msgs ...proto.Message) []*gogo.Any {
	result := make([]*gogo.Any, len(msgs))
	for i, msg := range msgs {
		result[i] = FromProtoMsgToAny(msg)
	}
	return result
}
