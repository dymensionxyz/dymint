package block_test

import (
	"testing"

	rdktypes "github.com/dymensionxyz/dymension-rdk/x/sequencers/types"
	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/block"
	sequencertypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
)

func TestConsensusMsgSigner(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		addr, err := block.ConsensusMsgSigner(new(rdktypes.ConsensusMsgUpsertSequencer))
		require.NoError(t, err)
		require.Equal(t, "cosmos1k3m3ck7fkza9zlwn95p0m73wugf5x4hxf83rqd", addr.String())
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := block.ConsensusMsgSigner(new(sequencertypes.MsgCreateSequencer))
		require.Error(t, err)
	})
}
