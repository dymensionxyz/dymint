package p2p_test

import (
	"context"
	"testing"

	"github.com/dymensionxyz/dymint/p2p"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
)

func TestBlockSync(t *testing.T) {

	logger := log.TestingLogger()
	ctx := context.Background()

	manager, err := testutil.GetManager(testutil.GetManagerConfig(), nil, nil, 1, 1, 0, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// required for tx validator
	assertRecv := func(tx *p2p.GossipMessage) bool {
		return true
	}

	// Create a block for height 1
	blocks, err := testutil.GenerateBlocksWithTxs(1, 1, manager.ProposerKey, 1)
	require.NoError(t, err)

	// Create commit
	commits, err := testutil.GenerateCommits(blocks, manager.ProposerKey)
	require.NoError(t, err)

	gossipedBlock := p2p.P2PBlock{Block: *blocks[0], Commit: *commits[0]}
	gossipedBlockbytes, err := gossipedBlock.MarshalBinary()
	require.NoError(t, err)

	// validators required
	validators := []p2p.GossipValidator{assertRecv, assertRecv, assertRecv, assertRecv, assertRecv}

	clients := testutil.StartTestNetwork(ctx, t, 1, map[int]testutil.HostDescr{
		0: {Conns: []int{}, ChainID: "1"},
	}, validators, logger)

	blocksync, err := p2p.StartBlockSync(ctx, clients[0].Host, datastore.NewMapDatastore(), nil, logger)
	require.NoError(t, err)

	//add block to blocksync protocol client 0
	cid, err := blocksync.AddBlock(ctx, gossipedBlockbytes)
	require.NoError(t, err)

	//add block to blocksync protocol client 0
	block, err := blocksync.GetBlock(ctx, cid)
	require.NoError(t, err)
	require.Equal(t, gossipedBlock, block)
}
