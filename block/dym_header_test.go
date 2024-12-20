package block

import (
	"testing"

	sequencers "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/dymensionxyz/dymint/types"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
)

func TestFoo(t *testing.T) {
	queue := NewConsensusMsgQueue()
	queue.Add(&sequencers.MsgUpgradeDRS{DrsVersion: 1})
	asAny := protoutils.FromProtoMsgSliceToAnySlice(queue.Get()...)
	h := types.MakeDymHeader(asAny)
	t.Logf("hash: %x", h.Hash())
	h = types.MakeDymHeader(nil)
	t.Logf("hash: %x", h.Hash())
}
