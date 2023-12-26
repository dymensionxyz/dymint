package types

import (
	"context"

	"github.com/rollkit/celestia-openrpc/types/blob"
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
}
