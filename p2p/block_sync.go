package p2p

import (
	"context"

	"github.com/dymensionxyz/dymint/types"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"

	mh "github.com/multiformats/go-multihash"

	"github.com/ipfs/boxo/bitswap/client"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/bitswap/server"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/go-cid"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
)

// Blocksync is a protocol used to retrieve blocks on demand from the P2P network.
// Nodes store received blocks from gossip in an IPFS blockstore and nodes are able to request them on demand using bitswap protocol.
// In order to discover the identifier (CID) of each block a DHT request needs to be made for the specific block height.
// Nodes need to advertise CIDs/height map to the DHT periodically.
// https://www.notion.so/dymension/ADR-x-Rollapp-block-sync-protocol-6ee48b232a6a45e09989d67f1a6c0297?pvs=4
type BlockSync struct {
	// service that reads/writes blocks either from local datastore or the network
	bsrv blockservice.BlockService
	// local datastore for IPFS blocks
	bstore blockstore.Blockstore
	// protocol used to obtain blocks from the P2P network
	net network.BitSwapNetwork
	// used to retrieve/create merkle DAGs using block data
	dsrv BlockSyncDagService
	// used to define content identifiers
	cidBuilder cid.Builder
	logger     types.Logger

	// receives blocks obtained from the network
	msgHandler BlockSyncMessageHandler
}

type BlockSyncMessageHandler func(block *P2PBlock)

func SetupBlockSync(ctx context.Context, h host.Host, store datastore.Datastore, msgHandler BlockSyncMessageHandler, logger types.Logger) *BlockSync {
	ds := dsync.MutexWrap(store)

	bs := blockstore.NewBlockstore(ds)

	net := network.NewFromIpfsHost(h, &routinghelpers.Null{}, network.Prefix("/dymension/block-sync/"))
	server := server.New(
		ctx,
		net,
		bs,
		server.ProvideEnabled(false), // we don't provide blocks over DHT
		server.SetSendDontHaves(false),
	)

	bsclient := client.New(
		ctx,
		net,
		bs,
		client.SetSimulateDontHavesOnTimeout(false),
		client.WithBlockReceivedNotifier(server),
		client.WithoutDuplicatedBlockStats(),
	)
	net.Start(server, bsclient)
	bsrv := blockservice.New(bs, bsclient)
	blockSync := &BlockSync{
		bsrv:       bsrv,
		net:        net,
		bstore:     bs,
		dsrv:       NewDAGService(bsrv),
		msgHandler: msgHandler,
		cidBuilder: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  1,
		},
		logger: logger,
	}

	return blockSync
}

func (blocksync *BlockSync) AddBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	return blocksync.dsrv.AddBlock(ctx, block)
}

func (blocksync *BlockSync) GetBlock(ctx context.Context, cid cid.Cid) (P2PBlock, error) {
	blockBytes, err := blocksync.dsrv.GetBlock(ctx, cid)
	if err != nil {
		blocksync.logger.Error("GetBlock", "err", err)
	}
	var block P2PBlock
	if err := block.UnmarshalBinary(blockBytes); err != nil {
		blocksync.logger.Error("Deserialize gossiped block", "error", err)
	}
	blocksync.logger.Debug("Blocksync block received ", "cid", cid)
	return block, nil
}

func (blocksync *BlockSync) RemoveBlock(ctx context.Context, cid cid.Cid) error {
	return blocksync.dsrv.Remove(ctx, cid)
}
