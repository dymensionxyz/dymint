package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
)

// Client is the jsonrpc client
type Client struct {
	DA      DAAPI
	Headers HeadersAPI
	State   StateAPI
	closer  multiClientCloser
}

// Balance implements DAClient.
func (c Client) Balance(ctx context.Context) (*Balance, error) {
	return c.State.Balance(ctx)
}

// Commit implements DAClient.
func (c Client) Commit(ctx context.Context, blobs []Blob, namespace Namespace) ([]Commitment, error) {
	return c.DA.Commit(ctx, blobs, namespace)
}

// Get implements DAClient.
func (c Client) Get(ctx context.Context, ids []ID, namespace Namespace) ([]Blob, error) {
	return c.DA.Get(ctx, ids, namespace)
}

// GetByHeight implements DAClient.
func (c Client) GetByHeight(ctx context.Context, height uint64) (*ExtendedHeader, error) {
	return c.Headers.GetByHeight(ctx, height)
}

// GetIDs implements DAClient.
func (c Client) GetIDs(ctx context.Context, height uint64, namespace Namespace) (*GetIDsResult, error) {
	return c.DA.GetIDs(ctx, height, namespace)
}

// GetProofs implements DAClient.
func (c Client) GetProofs(ctx context.Context, ids []ID, namespace Namespace) ([]Proof, error) {
	return c.DA.GetProofs(ctx, ids, namespace)
}

// MaxBlobSize implements DAClient.
func (c Client) MaxBlobSize(ctx context.Context) (uint64, error) {
	return c.DA.MaxBlobSize(ctx)
}

// Submit implements DAClient.
func (c Client) Submit(ctx context.Context, blobs []Blob, gasPrice float64, namespace Namespace) ([]ID, error) {
	return c.DA.Submit(ctx, blobs, gasPrice, namespace)
}

// SubmitWithOptions implements DAClient.
func (c Client) SubmitWithOptions(ctx context.Context, blobs []Blob, gasPrice float64, namespace Namespace, options []byte) ([]ID, error) {
	return c.DA.SubmitWithOptions(ctx, blobs, gasPrice, namespace, options)
}

// Validate implements DAClient.
func (c Client) Validate(ctx context.Context, ids []ID, proofs []Proof, namespace Namespace) ([]bool, error) {
	return c.DA.Validate(ctx, ids, proofs, namespace)
}

// multiClientCloser is a wrapper struct to close clients across multiple namespaces.
type multiClientCloser struct {
	closers []jsonrpc.ClientCloser
}

// register adds a new closer to the multiClientCloser
func (m *multiClientCloser) register(closer jsonrpc.ClientCloser) {
	m.closers = append(m.closers, closer)
}

// closeAll closes all saved clients.
func (m *multiClientCloser) closeAll() {
	for _, closer := range m.closers {
		closer()
	}
}

// Close closes the connections to all namespaces registered on the staticClient.
func (c *Client) Close() {
	c.closer.closeAll()
}

// NewClient returns a DA backend based on the uri
// and auth token.
func NewClient(uri, token string) (DAClient, error) {
	var client DAClient

	authHeader := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}

	jsonrpcClient, err := newClient(context.Background(), uri, authHeader)
	if err != nil {
		return nil, err
	}
	client = *jsonrpcClient

	return client, nil
}

func newClient(ctx context.Context, addr string, authHeader http.Header) (*Client, error) {
	var multiCloser multiClientCloser
	var client Client
	errs := getKnownErrorsMapping()
	for name, module := range moduleMap(&client) {
		closer, err := jsonrpc.NewMergeClient(ctx, addr, name, []interface{}{module}, authHeader, jsonrpc.WithErrors(errs))
		if err != nil {
			return nil, err
		}
		multiCloser.register(closer)
	}

	return &client, nil
}

func moduleMap(client *Client) map[string]interface{} {
	// TODO: this duplication of strings many times across the codebase can be avoided with issue #1176
	return map[string]interface{}{
		"da":     &client.DA.Internal,
		"header": &client.Headers.Internal,
		"state":  &client.State.Internal,
	}
}

// getKnownErrorsMapping returns a mapping of known error codes to their corresponding error types.
func getKnownErrorsMapping() jsonrpc.Errors {
	errs := jsonrpc.NewErrors()
	errs.Register(jsonrpc.ErrorCode(CodeBlobNotFound), new(*ErrBlobNotFound))
	errs.Register(jsonrpc.ErrorCode(CodeBlobSizeOverLimit), new(*ErrBlobSizeOverLimit))
	errs.Register(jsonrpc.ErrorCode(CodeTxTimedOut), new(*ErrTxTimedOut))
	errs.Register(jsonrpc.ErrorCode(CodeTxAlreadyInMempool), new(*ErrTxAlreadyInMempool))
	errs.Register(jsonrpc.ErrorCode(CodeTxIncorrectAccountSequence), new(*ErrTxIncorrectAccountSequence))
	errs.Register(jsonrpc.ErrorCode(CodeTxTooLarge), new(*ErrTxTooLarge))
	errs.Register(jsonrpc.ErrorCode(CodeContextDeadline), new(*ErrContextDeadline))
	errs.Register(jsonrpc.ErrorCode(CodeFutureHeight), new(*ErrFutureHeight))
	return errs
}
