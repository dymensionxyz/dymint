package p2p

import (
	"bytes"
	"context"
	"errors"
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



func (bsDagService *BlockSyncDagService) SaveBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	blockReader := bytes.NewReader(block)

	splitter := chunker.NewSizeSplitter(blockReader, chunker.DefaultBlockSize)
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
		err = protoNode.SetCidBuilder(bsDagService.cidBuilder)
		if err != nil {
			return cid.Undef, err
		}
		nodes = append(nodes, protoNode)

	}

	
	root := dag.NodeWithData(nil)
	err := root.SetCidBuilder(bsDagService.cidBuilder)
	if err != nil {
		return cid.Undef, err
	}

	
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
	err = bsDagService.Add(ctx, root)
	if err != nil {
		return cid.Undef, err
	}

	return root.Cid(), nil
}


func (bsDagService *BlockSyncDagService) LoadBlock(ctx context.Context, cid cid.Cid) ([]byte, error) {
	
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

func (bsDagService *BlockSyncDagService) DeleteBlock(ctx context.Context, cid cid.Cid) error {
	
	root, err := bsDagService.Get(ctx, cid)
	if err != nil {
		return err
	}

	
	for _, l := range root.Links() {
		err := bsDagService.Remove(ctx, l.Cid)
		if err != nil {
			return err
		}
	}
	return nil
}


func dagReader(root ipld.Node, ds ipld.DAGService) (io.Reader, error) {
	ctx := context.Background()
	buf := new(bytes.Buffer)

	
	for _, l := range root.Links() {
		n, err := ds.Get(ctx, l.Cid)
		if err != nil {
			return nil, err
		}
		rawdata, ok := n.(*dag.ProtoNode)
		if !ok {
			return nil, errors.New("read block DAG")
		}

		_, err = buf.Write(rawdata.Data())
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}
