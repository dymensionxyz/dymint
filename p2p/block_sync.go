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

type BlockSync struct {
	bsrv blockservice.BlockService

	bstore blockstore.Blockstore

	net network.BitSwapNetwork

	dsrv BlockSyncDagService

	cidBuilder cid.Builder
	logger     types.Logger
}

type BlockSyncMessageHandler func(block *BlockData)

func SetupBlockSync(ctx context.Context, h host.Host, store datastore.Datastore, logger types.Logger) *BlockSync {
	ds := dsync.MutexWrap(store)

	bs := blockstore.NewBlockstore(ds)

	bsnet := network.NewFromIpfsHost(h, &routinghelpers.Null{}, network.Prefix("/dymension/block-sync/"))

	bsserver := server.New(
		ctx,
		bsnet,
		bs,
		server.ProvideEnabled(false),
		server.SetSendDontHaves(false),
	)

	bsclient := client.New(
		ctx,
		bsnet,
		bs,
		client.SetSimulateDontHavesOnTimeout(false),
		client.WithBlockReceivedNotifier(bsserver),
		client.WithoutDuplicatedBlockStats(),
	)

	bsnet.Start(bsserver, bsclient)

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

func (blocksync *BlockSync) SaveBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	return blocksync.dsrv.SaveBlock(ctx, block)
}

func (blocksync *BlockSync) LoadBlock(ctx context.Context, cid cid.Cid) (BlockData, error) {
	blockBytes, err := blocksync.dsrv.LoadBlock(ctx, cid)
	if err != nil {
		return BlockData{}, err
	}
	var block BlockData
	if err := block.UnmarshalBinary(blockBytes); err != nil {
		return BlockData{}, fmt.Errorf("deserialize blocksync block %w", err)
	}
	return block, nil
}

func (blocksync *BlockSync) DeleteBlock(ctx context.Context, cid cid.Cid) error {
	return blocksync.dsrv.DeleteBlock(ctx, cid)
}
