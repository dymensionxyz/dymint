package p2p

import (
	"bytes"
	"context"
	"fmt"
	"io"

	unixfile "github.com/ipfs/boxo/ipld/unixfs/file"

	"github.com/ipfs/boxo/bitswap/client"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/bitswap/server"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	chunker "github.com/ipfs/boxo/chunker"
	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/boxo/ipld/unixfs/importer/balanced"
	uih "github.com/ipfs/boxo/ipld/unixfs/importer/helpers"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multicodec"
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
		bsrv:   bsrv,
		net:    net,
		bstore: bs,
	}

	//blockSync.DAGService = merkledag.NewDAGService(bsrv)
	blockSync.session = *blockservice.NewSession(ctx, bsrv)
	return blockSync, nil
}

func (blocksync *BlockSync) AddBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	/*b, err := blocks.NewBlockWithCid(block, c)
	if err != nil {
		return err
	}
	fmt.Println("Adding block", b.Cid())
	return blocksync.bstore.Put(ctx, b)*/

	//blocks.NewBlockWithCid()

	blockReader := bytes.NewReader(block)
	dsrv := merkledag.NewDAGService(blocksync.bsrv)
	//dsrv.Add()
	// Create a UnixFS graph from our file, parameters described here but can be visualized at https://dag.ipfs.tech/
	ufsImportParams := uih.DagBuilderParams{
		Maxlinks:  uih.DefaultLinksPerBlock, // Default max of 174 links per block
		RawLeaves: true,                     // Leave the actual file bytes untouched instead of wrapping them in a dag-pb protobuf wrapper
		CidBuilder: cid.V1Builder{ // Use CIDv1 for all links
			Codec:    uint64(multicodec.DagPb),
			MhType:   uint64(multicodec.Sha2_256), // Use SHA2-256 as the hash function
			MhLength: -1,                          // Use the default hash length for the given hash function (in this case 256 bits)
		},
		Dagserv: dsrv,
		NoCopy:  false,
	}
	ufsBuilder, err := ufsImportParams.New(chunker.NewSizeSplitter(blockReader, int64(chunker.DefaultBlockSize))) // Split the file up into fixed sized 256KiB chunks
	if err != nil {
		return cid.Undef, err
	}
	nd, err := balanced.Layout(ufsBuilder) // Arrange the graph with a balanced layout
	if err != nil {
		return cid.Undef, err
	}
	return nd.Cid(), nil
}

func (blocksync *BlockSync) GetBlock(ctx context.Context, cid cid.Cid) (blocks.Block, error) {
	//return blocksync.bsrv.GetBlock(ctx, cid)
	return blocksync.session.GetBlock(ctx, cid)
	//return blocksync.bstore.Get(ctx, block)
}
func (blocksync *BlockSync) GetBlocks(ctx context.Context, cids []cid.Cid) <-chan blocks.Block {
	return blocksync.session.GetBlocks(ctx, cids)
}

func (blockSync *BlockSync) ReceivedBlocks(p peer.ID, blocks []blocks.Block) {
	fmt.Println("Block received ", p, blocks)
}

func (blockSync *BlockSync) GetFile(ctx context.Context, cid cid.Cid) ([]byte, error) {
	//return blocksync.bsrv.GetBlock(ctx, cid)
	dserv := merkledag.NewReadOnlyDagService(merkledag.NewSession(ctx, merkledag.NewDAGService(blockSync.bsrv)))
	nd, err := dserv.Get(ctx, cid)
	if err != nil {
		return nil, err
	}

	unixFSNode, err := unixfile.NewUnixfsFile(ctx, dserv, nd)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if f, ok := unixFSNode.(files.File); ok {
		if _, err := io.Copy(&buf, f); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
