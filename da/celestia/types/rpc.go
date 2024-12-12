package types

import (
	"context"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/header"
	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/celestiaorg/celestia-openrpc/types/state"
)

type CelestiaRPCClient interface {
	/* ---------------------------------- blob ---------------------------------- */
	Get(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Blob, error)
	GetAll(context.Context, uint64, []share.Namespace) ([]*blob.Blob, error)
	GetProof(ctx context.Context, height uint64, namespace share.Namespace, commitment blob.Commitment) (*blob.Proof, error)
	Included(ctx context.Context, height uint64, namespace share.Namespace, proof *blob.Proof, commitment blob.Commitment) (bool, error)
	Submit(ctx context.Context, blobs []*blob.Blob, options *blob.SubmitOptions) (uint64, error)

	/* --------------------------------- header --------------------------------- */
	GetByHeight(ctx context.Context, height uint64) (*header.ExtendedHeader, error)

	/* ---------------------------------- state --------------------------------- */
	GetSignerBalance(ctx context.Context) (*state.Balance, error)
}
