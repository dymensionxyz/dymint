package p2p_test

import (
	"context"
	"testing"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/version"
	"github.com/ipfs/go-datastore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestBlockSync(t *testing.T) {
	version.DRS = "0"
	logger := log.TestingLogger()
	ctx := context.Background()

	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, 1, 1, 0, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, manager)

	// required for tx validator
	assertRecv := func(tx *p2p.GossipMessage) pubsub.ValidationResult {
		return pubsub.ValidationAccept
	}

	// Create a block for height 1
	blocks, err := testutil.GenerateBlocksWithTxs(1, 1, manager.LocalKey, 1, "test")
	require.NoError(t, err)

	// Create commit
	commits, err := testutil.GenerateCommits(blocks, manager.LocalKey)
	require.NoError(t, err)

	gossipedBlock := p2p.BlockData{Block: *blocks[0], Commit: *commits[0]}
	gossipedBlockbytes, err := gossipedBlock.MarshalBinary()
	require.NoError(t, err)

	// validators required
	validators := []p2p.GossipValidator{assertRecv, assertRecv, assertRecv, assertRecv, assertRecv}

	clients := testutil.StartTestNetwork(ctx, t, 1, map[int]testutil.HostDescr{
		0: {Conns: []int{}, ChainID: "1"},
	}, validators, logger)

	blocksync := p2p.SetupBlockSync(ctx, clients[0].Host, datastore.NewMapDatastore(), logger)
	require.NoError(t, err)

	// add block to blocksync protocol client 0
	cid, err := blocksync.SaveBlock(ctx, gossipedBlockbytes)
	require.NoError(t, err)

	// get block
	block, err := blocksync.LoadBlock(ctx, cid)
	require.NoError(t, err)
	require.Equal(t, gossipedBlock, block)

	// remove block
	err = blocksync.DeleteBlock(ctx, cid)
	require.NoError(t, err)
}
