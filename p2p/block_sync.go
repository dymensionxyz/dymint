package p2p

import (
	"context"

	"github.com/ipfs/boxo/bitswap/client"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/bitswap/server"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
)

type BlockSync struct {
	bsrv    blockservice.BlockService
	bstore  blockstore.Blockstore
	net     network.BitSwapNetwork
	session blockservice.Session
	//ipld.DAGService // become a DAG service
}

func StartBlockSync(ctx context.Context, h host.Host) (*BlockSync, error) {

	ds := dsync.MutexWrap(datastore.NewMapDatastore())
	bs := blockstore.NewBlockstore(ds)
	bs = blockstore.NewIdStore(bs) // handle identity multihashes, these don't require doing any actual lookups

	net := network.NewFromIpfsHost(h, &routinghelpers.Null{}, network.Prefix("/dymension/block-sync/"))
	server := server.New(
		ctx,
		net,
		bs,
		server.ProvideEnabled(false), // we don't provide blocks over DHT
		server.SetSendDontHaves(false),
	)

	client := client.New(
		ctx,
		net,
		bs,
		client.WithBlockReceivedNotifier(server),
		client.SetSimulateDontHavesOnTimeout(false),
		client.WithoutDuplicatedBlockStats(),
	)
	net.Start(server, client)
	bsrv := blockservice.New(bs, client)
	blockSync := &BlockSync{
		bsrv:   bsrv,
		net:    net,
		bstore: bs,
	}
	//blockSync.DAGService = merkledag.NewDAGService(bsrv)
	blockSync.session = *blockservice.NewSession(ctx, bsrv)
	return blockSync, nil
}

func (blocksync *BlockSync) AddBlock(ctx context.Context, block []byte) error {
	cid := blocks.NewBlock(block)
	return blocksync.bstore.Put(ctx, cid)

}

func (blocksync *BlockSync) GetBlock(ctx context.Context, block cid.Cid) (blocks.Block, error) {
	return blocksync.session.GetBlock(ctx, block)
	//return blocksync.bstore.Get(ctx, block)
}
