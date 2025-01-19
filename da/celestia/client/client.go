package daclient

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
func NewClient(uri, token string) (Client, error) {

	var client Client

	authHeader := http.Header{"Authorization": []string{fmt.Sprintf("Bearer %s", token)}}

	jsonrpcClient, err := newClient(context.Background(), uri, authHeader)
	if err != nil {
		return Client{}, err
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
