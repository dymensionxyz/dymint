package celestia

import (
	"context"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/share"
	"github.com/rollkit/celestia-openrpc/types/state"

	openrpc "github.com/rollkit/celestia-openrpc"

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

// SubmitPayForBlob submits a pay for blob transaction.
func (c *OpenRPC) SubmitPayForBlob(
	ctx context.Context,
	fee state.Int,
	gasLim uint64,
	blobs []*blob.Blob,
) (*state.TxResponse, error) {
	return c.rpc.State.SubmitPayForBlob(ctx, fee, gasLim, blobs)
}

// GetAll gets all blobs.
func (c *OpenRPC) GetAll(ctx context.Context, height uint64, namespaces []share.Namespace) ([]*blob.Blob, error) {
	return c.rpc.Blob.GetAll(ctx, height, namespaces)
}
