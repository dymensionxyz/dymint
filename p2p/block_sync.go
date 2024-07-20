package p2p

import (
	"context"
	"fmt"

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
	// service that reads/writes blocks either from local datastore or the P2P network
	bsrv blockservice.BlockService
	// local datastore for IPFS blocks
	bstore blockstore.Blockstore
	// protocol used to obtain blocks from the P2P network
	net network.BitSwapNetwork
	// used to find all data chunks that are part oth e same block
	dsrv BlockSyncDagService
	// used to define the content identifiers of each data chunk
	cidBuilder cid.Builder
	logger     types.Logger
}

type BlockSyncMessageHandler func(block *P2PBlockEvent)

// SetupBlockSync initializes all services required to provide and retrieve block data in the P2P network.
func SetupBlockSync(ctx context.Context, h host.Host, store datastore.Datastore, logger types.Logger) *BlockSync {

	//It constructs a datastore.
	ds := dsync.MutexWrap(store)

	//It sets a blockstore (to store IPFS data chunks) with the previous datastore.
	bs := blockstore.NewBlockstore(ds)

	//It initialized bitswap network used to retrieve data chunks from other peers in the P2P network https://docs.ipfs.tech/concepts/bitswap/
	bsnet := network.NewFromIpfsHost(h, &routinghelpers.Null{}, network.Prefix("/dymension/block-sync/"))

	//Bitswap server that provides data to the network.
	bsserver := server.New(
		ctx,
		bsnet,
		bs,
		server.ProvideEnabled(false), // we don't provide blocks over DHT
		server.SetSendDontHaves(false),
	)

	//Bitswap client that retrieves data from the network.
	bsclient := client.New(
		ctx,
		bsnet,
		bs,
		client.SetSimulateDontHavesOnTimeout(false),
		client.WithBlockReceivedNotifier(bsserver),
		client.WithoutDuplicatedBlockStats(),
	)

	//Bitswap network start.
	bsnet.Start(bsserver, bsclient)

	//BlockService with given datastore instance.
	bsrv := blockservice.New(bs, bsclient)

	blockSync := &BlockSync{
		bsrv:   bsrv,
		net:    bsnet,
		bstore: bs,
		dsrv:   NewDAGService(bsrv),
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

// AddBlock stores the blocks produced in the DAG services to be retrievable from the P2P network.
func (blocksync *BlockSync) AddBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	return blocksync.dsrv.AddBlock(ctx, block)
}

// GetBlock retrieves the blocks (from the local blockstore or the network) using the DAGService to discover all data chunks that are part of the same block.
func (blocksync *BlockSync) GetBlock(ctx context.Context, cid cid.Cid) (P2PBlockEvent, error) {
	blockBytes, err := blocksync.dsrv.GetBlock(ctx, cid)
	if err != nil {
		return P2PBlockEvent{}, err
	}
	var block P2PBlockEvent
	if err := block.UnmarshalBinary(blockBytes); err != nil {
		return P2PBlockEvent{}, fmt.Errorf("deserialize blocksync block %w", err)
	}
	return block, nil
}

// RemoveBlock removes the block from the DAGservice.
func (blocksync *BlockSync) RemoveBlock(ctx context.Context, cid cid.Cid) error {
	return blocksync.dsrv.Remove(ctx, cid)
}
