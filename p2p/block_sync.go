package p2p

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/dymensionxyz/dymint/types"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"

	chunker "github.com/ipfs/boxo/chunker"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	mh "github.com/multiformats/go-multihash"

	"github.com/ipfs/boxo/bitswap/client"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/bitswap/server"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
)

type BlockSync struct {
	bsrv       blockservice.BlockService
	bstore     blockstore.Blockstore
	net        network.BitSwapNetwork
	session    blockservice.Session
	dsrv       ipld.DAGService
	cidBuilder cid.Builder
	logger     types.Logger
}

func StartBlockSync(ctx context.Context, h host.Host, store datastore.Datastore, logger types.Logger) (*BlockSync, error) {

	//bs := blockstore.NewBlockstore(p.Store)
	//db, err := badger.Open(badger.DefaultOptions("/"))

	//d, err := leveldb.NewDatastore(path, &leveldb.Options{})
	/*if err != nil {
		return err
	}
	//ds := dsync.MutexWrap(db)
	d, err := leveldb.NewDatastore(path+"/data/p2p", &leveldb.Options{})
	if err != nil {
		return nil, err
	}*/
	ds := dsync.MutexWrap(store)
	//r.ds = syncds.MutexWrap(d)
	//ds := dsync.MutexWrap(datastore.NewMapDatastore())
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
		dsrv:   merkledag.NewDAGService(bsrv),

		cidBuilder: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  1,
		},
		logger: logger,
	}

	blockSync.session = *blockservice.NewSession(ctx, bsrv)
	return blockSync, nil
}

func (blocksync *BlockSync) AddBlock(ctx context.Context, height uint64, block []byte) (cid.Cid, error) {

	blockReader := bytes.NewReader(block)

	splitter := chunker.NewSizeSplitter(blockReader, int64(chunker.DefaultBlockSize))
	nodes := []*dag.ProtoNode{}

	for {
		nextData, err := splitter.NextBytes()
		if err == io.EOF {
			break
		}
		if err != nil {
			return cid.Undef, err
		}
		protoNode := dag.NodeWithData(nextData)
		protoNode.SetCidBuilder(blocksync.cidBuilder)

		nodes = append(nodes, protoNode)

	}

	root := dag.NodeWithData(nil)
	root.SetCidBuilder(blocksync.cidBuilder)
	for _, n := range nodes {

		err := root.AddNodeLink(n.Cid().String(), n)
		if err != nil {
			return cid.Undef, err
		}
		err = blocksync.dsrv.Add(ctx, n)
		if err != nil {
			return cid.Undef, err
		}
	}
	err := blocksync.dsrv.Add(ctx, root)
	if err != nil {
		return cid.Undef, err
	}

	return root.Cid(), nil

}

func (blocksync *BlockSync) GetBlock(ctx context.Context, blockId string) ([]byte, error) {

	cid := cid.MustParse(blockId)
	nd, err := blocksync.dsrv.Get(ctx, cid)
	//nd, err := dserv.Get(ctx, cid)
	if err != nil {
		return nil, err
	}

	read, err := dagReader(nd, blocksync.dsrv)
	if err != nil {
		return nil, err
	}
	datagot, err := io.ReadAll(read)
	if err != nil {
		return nil, err
	}
	return datagot, nil
	//return blocksync.bstore.Get(ctx, block)
}

// makeTestDAGReader takes the root node as returned by makeTestDAG and
// provides a reader that reads all the RawData from that node and its children.
func dagReader(root ipld.Node, ds ipld.DAGService) (io.Reader, error) {
	ctx := context.Background()
	buf := new(bytes.Buffer)
	fmt.Println("Reading ", string(root.RawData()))
	//buf.Write(root.RawData())
	for _, l := range root.Links() {
		n, err := ds.Get(ctx, l.Cid)
		if err != nil {
			return nil, err
		}
		rawdata, ok := n.(*dag.ProtoNode)
		if !ok {
			return nil, err
		}
		fmt.Println("Reading ", string(rawdata.Data()))

		_, err = buf.Write(rawdata.Data())
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}
