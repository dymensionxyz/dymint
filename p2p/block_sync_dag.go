package p2p

import (
	"bytes"
	"context"
	"fmt"
	"io"

	chunker "github.com/ipfs/boxo/chunker"
	mh "github.com/multiformats/go-multihash"

	"github.com/ipfs/boxo/blockservice"
	"github.com/ipfs/boxo/ipld/merkledag"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

type BlockSyncDagService struct {
	ipld.DAGService
	cidBuilder cid.Builder
}

func NewDAGService(bsrv blockservice.BlockService) BlockSyncDagService {

	bsDagService := &BlockSyncDagService{
		cidBuilder: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  1,
		},
	}
	bsDagService.DAGService = merkledag.NewDAGService(bsrv)

	return *bsDagService
}

func (bsDagService *BlockSyncDagService) AddBlock(ctx context.Context, block []byte) (cid.Cid, error) {

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
		protoNode.SetCidBuilder(bsDagService.cidBuilder)

		nodes = append(nodes, protoNode)

	}

	root := dag.NodeWithData(nil)
	root.SetCidBuilder(bsDagService.cidBuilder)
	for _, n := range nodes {

		err := root.AddNodeLink(n.Cid().String(), n)
		if err != nil {
			return cid.Undef, err
		}
		err = bsDagService.Add(ctx, n)
		if err != nil {
			return cid.Undef, err
		}
	}
	err := bsDagService.Add(ctx, root)
	if err != nil {
		return cid.Undef, err
	}

	return root.Cid(), nil
}

func (bsDagService *BlockSyncDagService) GetBlock(ctx context.Context, blockId string) ([]byte, error) {

	cid := cid.MustParse(blockId)
	nd, err := bsDagService.Get(ctx, cid)
	if err != nil {
		return nil, err
	}

	read, err := dagReader(nd, bsDagService)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(read)
	if err != nil {
		return nil, err
	}
	return data, nil
}

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
