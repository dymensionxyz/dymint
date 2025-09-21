package p2p

import (
	"bytes"
	"context"
	"errors"
	"io"

	chunker "github.com/ipfs/boxo/chunker"
	mh "github.com/multiformats/go-multihash"

	"github.com/ipfs/boxo/blockservice"
	dag "github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-cid"
	ipld "github.com/ipfs/go-ipld-format"
)

type BlockSyncDagService struct {
	ipld.DAGService
	cidBuilder cid.Builder
}

// NewDAGService inits the DAGservice used to retrieve/send blocks data in the P2P.
// Block data is organized in a merkle DAG using IPLD (https://ipld.io/docs/)
func NewDAGService(bsrv blockservice.BlockService) BlockSyncDagService {
	bsDagService := &BlockSyncDagService{
		cidBuilder: &cid.Prefix{
			Codec:    cid.DagProtobuf,
			MhLength: -1,
			MhType:   mh.SHA2_256,
			Version:  1,
		},
	}
	bsDagService.DAGService = dag.NewDAGService(bsrv)

	return *bsDagService
}

// SaveBlock splits the block in chunks of 256KB and it creates a new merkle DAG with them. it returns the content identifier (cid) of the root node of the DAG.
// Using the root CID the whole block can be retrieved using the DAG service
func (bsDagService *BlockSyncDagService) SaveBlock(ctx context.Context, block []byte) (cid.Cid, error) {
	blockReader := bytes.NewReader(block)

	splitter := chunker.NewSizeSplitter(blockReader, chunker.DefaultBlockSize)
	nodes := []*dag.ProtoNode{}

	// the loop creates nodes for each block chunk and sets each cid
	for {
		nextData, err := splitter.NextBytes()
		if errors.Is(err, io.EOF) {
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

	// an empty root node is created
	root := dag.NodeWithData(nil)
	err := root.SetCidBuilder(bsDagService.cidBuilder)
	if err != nil {
		return cid.Undef, err
	}

	// and linked to all chunks that are added to the DAGservice
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

// LoadBlock returns the block data obtained from the DAGService, using the root cid, either from the network or the local blockstore
func (bsDagService *BlockSyncDagService) LoadBlock(ctx context.Context, cid cid.Cid) ([]byte, error) {
	// first it gets the root node
	nd, err := bsDagService.Get(ctx, cid)
	if err != nil {
		return nil, err
	}

	// then it gets all the data from the root node
	read, err := dagReader(nd, bsDagService)
	if err != nil {
		return nil, err
	}

	// the data is read to bytes array
	data, err := io.ReadAll(read)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (bsDagService *BlockSyncDagService) DeleteBlock(ctx context.Context, cid cid.Cid) error {
	// first it gets the root node
	root, err := bsDagService.Get(ctx, cid)
	if err != nil {
		return err
	}

	// then it iterates all the cids to remove them from the block store
	for _, l := range root.Links() {
		err := bsDagService.Remove(ctx, l.Cid)
		if err != nil {
			return err
		}
	}
	return nil
}

// dagReader is used to read the DAG (all the block chunks) from the root (IPLD) node
func dagReader(root ipld.Node, ds ipld.DAGService) (io.Reader, error) {
	ctx := context.Background()
	buf := new(bytes.Buffer)

	// the loop retrieves all the nodes (block chunks) either from the local store or the network, in case it is not there.
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
