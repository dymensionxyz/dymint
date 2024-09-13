package celestia

import (
	"context"

	openrpc "github.com/celestiaorg/celestia-openrpc"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/celestia-openrpc/types/share"

	"github.com/dymensionxyz/dymint/da/celestia/types"
)

var _ types.CelestiaRPCClient = &OpenRPC{}

// OpenRPC is a wrapper around the openrpc client.
type OpenRPC struct {
	rpc *openrpc.Client
}

// NewOpenRPC creates a new openrpc client.
func NewOpenRPC(rpc *openrpc.Client) *OpenRPC {
	return &OpenRPC{
		rpc: rpc,
	}
}

// GetAll gets all blobs.
func (c *OpenRPC) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return c.rpc.Blob.GetAll(ctx, height, namespaces)
}

// Submit blobs.
func (c *OpenRPC) Submit(ctx context.Context, blobs []*blob.Blob, options *blob.SubmitOptions) (uint64, error) {
	return c.rpc.Blob.Submit(ctx, blobs, options)
}

// Getting proof for submitted blob
func (c *OpenRPC) GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error) {
	return c.rpc.Blob.GetProof(ctx, height, namespace, commitment)
}

// Get blob for a specific share commitment
func (c *OpenRPC) Get(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Blob, error) {
	return c.rpc.Blob.Get(ctx, height, namespace, commitment)
}

// Get extended Celestia headers for a specific height
func (c *OpenRPC) GetByHeight(ctx context.Context, height uint64) (*header.ExtendedHeader, error) {
	return c.rpc.Header.GetByHeight(ctx, height)
}

// Get extended Celestia headers for a specific height
func (c *OpenRPC) Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error) {
	return c.rpc.Blob.Included(ctx, height, namespace, proof, commitment)
}
