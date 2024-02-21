package types

import (
	"context"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/header"
	"github.com/rollkit/celestia-openrpc/types/share"
	"github.com/rollkit/celestia-openrpc/types/state"
)

type CelestiaRPCClient interface {
	SubmitPayForBlob(
		ctx context.Context,
		fee state.Int,
		gasLim uint64,
		blobs []*blob.Blob,
	) (*state.TxResponse, error)
	GetAll(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)
	Submit(ctx context.Context, blobs []*blob.Blob, options *openrpc.SubmitOptions) (uint64, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Get(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Blob, error)
	GetHeaders(ctx context.Context, height uint64) (*header.ExtendedHeader, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
}
