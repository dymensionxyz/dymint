package types

import (
	"context"

	openrpc "github.com/rollkit/celestia-openrpc"

	"github.com/rollkit/celestia-openrpc/types/blob"
	"github.com/rollkit/celestia-openrpc/types/header"
	"github.com/rollkit/celestia-openrpc/types/share"
)

type CelestiaRPCClient interface {
	/* ---------------------------------- blob ---------------------------------- */
	Get(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Blob, error)
	GetAll(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
	Submit(ctx context.Context, blobs []*blob.Blob, options *openrpc.SubmitOptions) (uint64, error)

	/* --------------------------------- header --------------------------------- */
	GetByHeight(ctx context.Context, height uint64) (*header.ExtendedHeader, error)
}
