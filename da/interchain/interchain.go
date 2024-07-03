package interchain

import (
	"context"
	"encoding/json"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/tendermint/tendermint/libs/bytes"
	"github.com/tendermint/tendermint/libs/pubsub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

var (
	_ da.ClientV2         = &DALayerClient{}
	_ da.BatchRetrieverV2 = &DALayerClient{}
)

type DAClient interface {
	Context() sdkclient.Context
	BroadcastTx(string, ...sdk.Msg) (cosmosclient.Response, error)
	Params(context.Context) (interchainda.Params, error)
	Blob(ctx context.Context, id interchainda.BlobID) (*interchainda.QueryBlobResponse, error)
	GetTx(context.Context, string) (*tx.GetTxResponse, error)
	ABCIQueryWithProof(ctx context.Context, path string, data bytes.HexBytes, height int64) (*ctypes.ResultABCIQuery, error)
}

// DALayerClient is a client for DA-provider blockchains supporting the interchain-da module.
type DALayerClient struct {
	logger types.Logger
	ctx    context.Context
	cancel context.CancelFunc
	cdc    codec.Codec
	synced chan struct{}

	accountAddress string // address of the sequencer in the DA layer
	daClient       DAClient
	daConfig       DAConfig
}

// Init is called once. It reads the DA client configuration and initializes resources for the interchain DA provider.
func (c *DALayerClient) Init(rawConfig []byte, _ *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	ctx := context.Background()

	// Read DA layer config
	var config DAConfig
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	err = config.Verify()
	if err != nil {
		return fmt.Errorf("intechain DA config verification failed: %w", err)
	}

	// Create cosmos client with DA layer
	client, err := newDAClient(ctx, config)
	if err != nil {
		return fmt.Errorf("can't create DA layer client: %w", err)
	}

	// Query DA layer interchain-da module params
	daParams, err := client.Params(ctx)
	if err != nil {
		return fmt.Errorf("can't query DA layer interchain-da module params: %w", err)
	}
	config.DAParams = daParams

	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	interfaceRegistry.RegisterImplementations(&interchainda.MsgSubmitBlob{})
	cdc := codec.NewProtoCodec(interfaceRegistry)

	addr, err := client.Address(config.AccountName)
	if err != nil {
		return fmt.Errorf("cannot get '%s' account address from the provided keyring: %w", config.AccountName, err)
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)

	// Fill client fields
	c.logger = logger
	c.ctx = ctx
	c.cancel = cancel
	c.cdc = cdc
	c.synced = make(chan struct{})
	c.accountAddress = addr.String()
	c.daClient = client
	c.daConfig = config

	// Apply client options
	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start is called once, after Init. It starts the operation of DALayerClient, and Dymint will start submitting batches to the provider.
// It fetches the latest interchain module parameters and sets up a subscription to receive updates when the provider updates these parameters.
// This ensures that the client is always up-to-date.
func (c *DALayerClient) Start() error {
	// TODO: Setup a subscription to event EventUpdateParams
	return nil
}

// Stop is called once, when DALayerClient is no longer needed.
func (c *DALayerClient) Stop() error {
	c.cancel()
	return nil
}

// Synced returns channel for on sync event
func (c *DALayerClient) Synced() <-chan struct{} {
	return c.synced
}

func (c *DALayerClient) GetClientType() da.Client {
	return da.Interchain
}

func (c *DALayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	panic("CheckBatchAvailability method is not supported by the interchain DA clint")
}
