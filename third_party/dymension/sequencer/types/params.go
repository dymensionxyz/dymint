package types

import proto "github.com/cosmos/gogoproto/proto"

func (m *Params) String() string {
	return proto.CompactTextString(m)
}
