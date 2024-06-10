package p2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"

	chunker "github.com/ipfs/boxo/chunker"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	mh "github.com/multiformats/go-multihash"

	"github.com/ipfs/boxo/bitswap/client"
	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/boxo/bitswap/server"
	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/ipld/merkledag"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	ipld "github.com/ipfs/go-ipld-format"
	routinghelpers "github.com/libp2p/go-libp2p-routing-helpers"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type BlockSync struct {
	bsrv       blockservice.BlockService
	bstore     blockstore.Blockstore
	net        network.BitSwapNetwork
	session    blockservice.Session
	dsrv       ipld.DAGService
	cidBuilder cid.Builder
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
		dsrv:   merkledag.NewDAGService(bsrv),
		/*cidBuilder: &cidBuilder{
			Version:  0,
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
		},*/
		cidBuilder: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  1,
		},
	}

	//blockSync.DAGService = merkledag.NewDAGService(bsrv)
	blockSync.session = *blockservice.NewSession(ctx, bsrv)
	return blockSync, nil
}

func (blocksync *BlockSync) AddBlock(ctx context.Context, height uint64, block []byte) (cid.Cid, error) {
	/*b := blocks.NewBlock(block)
	fmt.Println("Adding block", b.Cid())
	return b.Cid(), blocksync.bstore.Put(ctx, b)*/

	//blocks.NewBlockWithCid()

	//blocksync.cidBuilder.(*cidBuilder).Height = height
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
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, height)
	b := buf[:n]
	root := dag.NodeWithData(b)
	root.SetCidBuilder(blocksync.cidBuilder)
	//root := dag.NodeWithData(nil)
	fmt.Println("Setcidbuilder", root.Cid())
	for _, n := range nodes {

		//fmt.Println("LINK", n.Cid())

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
	/*read, _ := dagReader(root, blocksync.dsrv)
	datagot, _ := io.ReadAll(read)
	fmt.Println("data sent", len(datagot))
	if err != nil {
		return cid.Undef, err
	}*/
	fmt.Println("Setcidbuilder", root.Cid())

	return root.Cid(), nil
	//dsrv.Add()
	// Create a UnixFS graph from our file, parameters described here but can be visualized at https://dag.ipfs.tech/
	/*ufsImportParams := uih.DagBuilderParams{
		Maxlinks:  uih.DefaultLinksPerBlock, // Default max of 174 links per block
		RawLeaves: true,                     // Leave the actual file bytes untouched instead of wrapping them in a dag-pb protobuf wrapper
		CidBuilder: cid.V1Builder{ // Use CIDv1 for all links
			Codec:    uint64(multicodec.DagPb),
			MhType:   uint64(multicodec.Sha2_256), // Use SHA2-256 as the hash function
			MhLength: -1,                          // Use the default hash length for the given hash function (in this case 256 bits)
		},
		Dagserv: blocksync.dsrv,
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
	blocksync.dsrv.Add()
	nd.Links()
	return nd.Cid(), nil*/
}

/*func (blocksync *BlockSync) GetBlock(ctx context.Context, cid cid.Cid) (blocks.Block, error) {
	//return blocksync.bsrv.GetBlock(ctx, cid)
	return blocksync.session.GetBlock(ctx, cid)
	//return blocksync.bstore.Get(ctx, block)
}*/

func (blocksync *BlockSync) GetBlocks(ctx context.Context, cids []cid.Cid) <-chan blocks.Block {
	return blocksync.session.GetBlocks(ctx, cids)
}

func (blockSync *BlockSync) ReceivedBlocks(p peer.ID, blocks []blocks.Block) {
	fmt.Println("Block received ", p, blocks)
}

func (blocksync *BlockSync) GetBlock(ctx context.Context, blockId string) ([]byte, error) {
	//return blocksync.bsrv.GetBlock(ctx, cid)
	//return blocksync.session.GetBlock(ctx, cid)
	/*blocksync.cidBuilder.(*cidBuilder).Height = height
	cid, err := blocksync.cidBuilder.Sum(nil)
	if err != nil {
		return nil, err
	}*/

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

func (blocksync *BlockSync) GetFile(ctx context.Context, cid cid.Cid) ([]byte, error) {
	//return blocksync.bsrv.GetBlock(ctx, cid)
	//dserv := merkledag.NewReadOnlyDagService(merkledag.NewSession(ctx, merkledag.NewDAGService(blockSync.bsrv)))
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
	/*unixFSNode, err := unixfile.NewUnixfsFile(ctx, blockSync.dsrv, nd)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if f, ok := unixFSNode.(files.File); ok {
		if _, err := io.Copy(&buf, f); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil*/
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

/*type idBuilder interface {
	cid.Builder
}
type cidBuilder struct {
	Version  uint64
	Codec    uint64
	MhType   uint64
	MhLength int
	Height   uint64
}*/

/*func (c *cidBuilder) Sum(data []byte) (cid.Cid, error) {
length := c.MhLength
if c.MhType == mh.IDENTITY {
	length = -1
}

if c.Version == 0 && (c.MhType != mh.SHA2_256 ||
	(c.MhLength != 32 && c.MhLength != -1)) {

	return cid.Undef, cid.ErrInvalidCid{fmt.Errorf("invalid v0 prefix")}
}

var hash mh.Multihash
var err error
var b []byte
//if len(data) == 0 {
buf := make([]byte, binary.MaxVarintLen64)
n := binary.PutUvarint(buf, c.Height)
b = buf[:n]
hash, err = mh.Sum(b, c.MhType, length)
/*} else {
	hash, err = mh.Sum(data, c.MhType, length)
}*/
/*if err != nil {
		return cid.Undef, cid.ErrInvalidCid{err}
	}
	fmt.Println("b", b)
	fmt.Println("data", data)

	fmt.Println("type", c.MhType)
	fmt.Println("length", length)

	fmt.Println("Cid", cid.NewCidV1(c.Codec, hash))
	switch c.Version {
	case 0:
		return cid.NewCidV0(hash), nil
	case 1:
		return cid.NewCidV1(c.Codec, hash), nil
	default:
		return cid.Undef, cid.ErrInvalidCid{fmt.Errorf("invalid cid version")}
	}
}*/

/*func (c *cidBuilder) Sum(data []byte) (cid.Cid, error) {
	mhLen := c.MhLength
	if mhLen <= 0 {
		mhLen = -1
	}
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, c.Height)
	b := buf[:n]
	hash, err := mh.Sum(b, c.MhType, mhLen)
	//hash, err := mh.Sum(data, c.MhType, mhLen)

	if err != nil {
		return cid.Undef, err
	}
	return cid.NewCidV0(hash), nil
}
func (c *cidBuilder) GetCodec() uint64 { return c.Codec }
func (c *cidBuilder) WithCodec(codec uint64) cid.Builder {
	if codec == c.Codec {
		return c
	}
	c.Codec = codec
	return c
}*/
