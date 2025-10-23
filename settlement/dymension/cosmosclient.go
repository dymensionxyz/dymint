package dymension

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	sequencertypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
)

// CosmosClient is an interface for interacting with cosmos client chains.
// It is a wrapper around the cosmos client in order to provide with an interface which can be implemented by
// other clients and can easily be mocked for testing purposes.
// Currently it contains only the methods that are used by the dymension hub client.
type CosmosClient interface {
	Context() sdkclient.Context
	StartEventListener() error
	StopEventListener() error
	EventListenerQuit() <-chan struct{}
	SubscribeToEvents(ctx context.Context, subscriber string, query string, outCapacity ...int) (out <-chan ctypes.ResultEvent, err error)
	UnsubscribeAll(ctx context.Context, subscriber string) error
	BroadcastTx(accountName string, msgs ...sdktypes.Msg) (cosmosclient.Response, error)
	GetRollappClient() rollapptypes.QueryClient
	GetSequencerClient() sequencertypes.QueryClient
	GetAccount(accountName string) (cosmosaccount.Account, error)
	GetBalance(ctx context.Context, accountName string, denom string) (*sdktypes.Coin, error)
	GetChainID() string
}

type cosmosClient struct {
	cosmosclient.Client
}

var _ CosmosClient = &cosmosClient{}

// NewCosmosClient creates a new cosmos client
func NewCosmosClient(client cosmosclient.Client) CosmosClient {
	return &cosmosClient{client}
}

func (c *cosmosClient) StartEventListener() error {
	return c.RPC.Start()
}

func (c *cosmosClient) StopEventListener() error {
	return c.RPC.Stop()
}

func (c *cosmosClient) EventListenerQuit() <-chan struct{} {
	return c.RPC.Quit()
}

func (c *cosmosClient) SubscribeToEvents(ctx context.Context, subscriber string, query string, outCapacity ...int) (out <-chan ctypes.ResultEvent, err error) {
	return c.WSEvents.Subscribe(ctx, subscriber, query, outCapacity...)
}

func (c *cosmosClient) UnsubscribeAll(ctx context.Context, subscriber string) error {
	return c.WSEvents.UnsubscribeAll(ctx, subscriber)
}

func (c *cosmosClient) GetRollappClient() rollapptypes.QueryClient {
	return rollapptypes.NewQueryClient(c.Context())
}

func (c *cosmosClient) GetSequencerClient() sequencertypes.QueryClient {
	return sequencertypes.NewQueryClient(c.Context())
}

func (c *cosmosClient) GetAccount(accountName string) (cosmosaccount.Account, error) {
	acc, err := c.AccountRegistry.GetByName(accountName)
	if err != nil {
		if strings.Contains(err.Error(), "too many failed passphrase attempts") {
			return cosmosaccount.Account{}, fmt.Errorf("account registry get by name: %w:%w", gerrc.ErrUnauthenticated, err)
		}
		var accNotExistErr *cosmosaccount.AccountDoesNotExistError
		if errors.As(err, &accNotExistErr) {
			return cosmosaccount.Account{}, fmt.Errorf("account registry get by name: %w:%w", gerrc.ErrNotFound, err)
		}
	}
	return acc, err
}

func (c *cosmosClient) GetBalance(ctx context.Context, address string, denom string) (*sdktypes.Coin, error) {
	balance, err := c.Balance(ctx, address, denom)
	if err != nil {
		return &sdktypes.Coin{}, err
	}

	return balance.Balance, nil
}

func (c *cosmosClient) GetChainID() string {
	return c.Client.Context().ChainID
}
