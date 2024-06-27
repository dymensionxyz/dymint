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
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

type DAClient interface {
	Context() sdkclient.Context
	BroadcastTx(accountName string, msgs ...sdk.Msg) (cosmosclient.Response, error)
	Params(ctx context.Context) (interchainda.Params, error)
}

// DataAvailabilityLayerClient is a client for DA-provider blockchains supporting the interchain-da module.
type DataAvailabilityLayerClient struct {
	logger types.Logger
	ctx    context.Context
	cancel context.CancelFunc
	cdc    codec.Codec
	synced chan struct{}

	pubsubServer *pubsub.Server

	daClient DAClient
	daConfig DAConfig
}

// Init is called once. It reads the DA client configuration and initializes resources for the interchain DA provider.
func (c *DataAvailabilityLayerClient) Init(rawConfig []byte, server *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	ctx := context.Background()

	// Read DA layer config
	var config DAConfig
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
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

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)

	// Create codec
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	interfaceRegistry.RegisterImplementations(&interchainda.MsgSubmitBlob{})
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Fill client fields
	c.logger = logger
	c.ctx = ctx
	c.cancel = cancel
	c.cdc = cdc
	c.synced = make(chan struct{})
	c.pubsubServer = server
	c.daClient = client
	c.daConfig = config

	// Apply client options
	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start is called once, after Init. It starts the operation of DataAvailabilityLayerClient, and Dymint will start submitting batches to the provider.
// It fetches the latest interchain module parameters and sets up a subscription to receive updates when the provider updates these parameters.
// This ensures that the client is always up-to-date.
func (c *DataAvailabilityLayerClient) Start() error {
	// Get the connectionID from the dymension hub for the da chain
	c.daConfig.ClientID = dymension.(c.chainConfig.ChainID)

	// Setup a subscription to event EventUpdateParams
	c.grpc.Subscribe(func() {
		// This event is thrown at the end of the block when the module params are updated
		if block.event == EventUpdateParams {
			// when the chain params are updated, update the client config to reflect the same
			da.chainConfig.chainParams = block.event.new_params
		}
	})
}

// Stop is called once, when DataAvailabilityLayerClient is no longer needed.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.pubsubServer.Stop()
	c.cancel()
	return nil
}

// Synced returns channel for on sync event
func (c *DataAvailabilityLayerClient) Synced() <-chan struct{} {
	return c.synced
}

func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Interchain
}

func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	result, err := c.submitBatch(batch)
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
			SubmitMetaData: nil,
		}
	}
	return da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Submission successful",
		},
		SubmitMetaData: &da.DASubmitMetaData{
			Height:     height,
			Namespace:  c.config.NamespaceID.Bytes(),
			Client:     da.Celestia,
			Commitment: commitment,
			Index:      0,
			Length:     0,
			Root:       nil,
		},
	}
}

type submitBatchResult struct {
	BlobID   uint64
	BlobHash string
}

func (c *DataAvailabilityLayerClient) submitBatch(batch *types.Batch) (submitBatchResult, error) {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return submitBatchResult{}, fmt.Errorf("can't marshal batch: %w", err)
	}

	if len(blob) > int(c.daConfig.DAParams.MaxBlobSize) {
		return submitBatchResult{}, fmt.Errorf("blob size %d exceeds the maximum allowed size %d", len(blob), c.daConfig.DAParams.MaxBlobSize)
	}

	feesToPay := sdk.NewCoin(c.daConfig.DAParams.CostPerByte.Denom, c.daConfig.DAParams.CostPerByte.Amount.MulRaw(int64(len(blob))))

	msg := interchainda.MsgSubmitBlob{
		Creator: c.daConfig.AccountName,
		Blob:    blob,
		Fees:    feesToPay,
	}

	txResp, err := c.daClient.BroadcastTx(c.daConfig.AccountName, &msg)
	if err != nil {
		return submitBatchResult{}, fmt.Errorf("can't broadcast MsgSubmitBlob to the DA layer: %w", err)
	}
	if txResp.Code != 0 {
		return submitBatchResult{}, fmt.Errorf("MsgSubmitBlob broadcast tx status code is not 0: code %d", txResp.Code)
	}

	var resp interchainda.MsgSubmitBlobResponse
	err = txResp.Decode(&resp)
	if err != nil {
		return submitBatchResult{}, fmt.Errorf("can't decode MsgSubmitBlob response: %w", err)
	}

	// trigger ibc stateupdate - optional (?)
	// other ibc interactions would trigger this anyway. But until then, inclusion cannot be verified.
	// better to trigger a stateupdate now imo
	dymension.tx.ibc.client.updatestate(c.daConfig.clientID) // could import the go relayer and execute their funcs

	return submitBatchResult{
		BlobID:   resp.BlobId,
		BlobHash: resp.BlobHash,
	}, nil
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	panic("implement me")
}
