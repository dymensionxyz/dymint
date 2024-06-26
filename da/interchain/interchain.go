package interchain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	interchaindatypes "github.com/dymensionxyz/interchain-da/x/interchain_da/types"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement/dymension"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
)

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// DataAvailabilityLayerClient is a client for DA-provider blockchains supporting the interchain-da module.
type DataAvailabilityLayerClient struct {
	logger types.Logger
	ctx    context.Context
	cancel context.CancelFunc
	cdc    codec.Codec
	synced chan struct{}

	pubsubServer    *pubsub.Server

	daClient       DAClient
	daConfig       DAConfig
	encodingConfig EncodingConfig // The DA chain's encoding config
}

// Init is called once. It reads the DA client configuration and initializes resources for the interchain DA provider.
func (c *DataAvailabilityLayerClient) Init(rawConfig []byte, server *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	var config DAConfig
	err := json.Unmarshal(rawConfig, &config)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	interchaindatypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)

	ctx, cancel := context.WithCancel(context.Background())

	c.logger = logger
	c.ctx = ctx
	c.cancel = cancel
	c.cdc = cdc
	c.synced = make(chan struct{})
	c.pubsubServer = server
	c.daConfig = config

	client, err := cosmosclient.New(ctx, getCosmosClientOptions(config)..., )
	if err != nil {
		return fmt.Errorf("can't create DA layer client: %w", err)
	}
	c.daClient = client

	for _, apply := range options {
		apply(c)
	}

	return nil
}

// Start is called once, after Init. It starts the operation of DataAvailabilityLayerClient, and Dymint will start submitting batches to the provider.
// It fetches the latest interchain module parameters and sets up a subscription to receive updates when the provider updates these parameters.
// This ensures that the client is always up-to-date.
func (c *DataAvailabilityLayerClient) Start() error {
	// Get the module parameters from the chain
	c.daConfig.ChainParams = c.grpc.GetModuleParams()

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
	c.daClient.Close()
	c.pubsubServer.Stop()
	c.cancel()
}

// Synced returns channel for on sync event
func (c *DataAvailabilityLayerClient) Synced() <-chan struct{} {
	return c.synced
}

func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Interchain
}

func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err},
		}
	}

	if len(blob) > int(c.daConfig.ChainParams.MaxBlobSize) {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err},
		}
	}

	// submit the blob to da chain

	// calculate the da fees to pay
	// feesToPay = params.CostPerByte * len(blob)
	feesToPay := sdk.NewCoin(c.daConfig.ChainParams.CostPerByte.Denom, c.daConfig.ChainParams.CostPerByte.Amount.MulRaw(int64(len(blob))))

	// generate the submit blob msg

	msg := interchaindatypes.MsgSubmitBlob{
		Creator: "",
		Blob:    blob,
		Fees:    feesToPay,
	}
	// use msg placeholder for now not to import the interchain-da module directly
	msgP := new(sdk.Msg)
	msg := *msgP

	// wrap the msg into a tx
	txBuilder := c.encodingConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err},
		}
	}

	ctx := context.Background()
	// sign and broadcast the tx
	txBytes, err := c.encodingConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err},
		}
	}

	txHash, err := c.txClient.BroadcastTx(ctx, &tx.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    tx.BroadcastMode_BROADCAST_MODE_SYNC,
	})
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err},
		}
	}

	c.txClient.GetTx(ctx, &tx.GetTxRequest{
		Hash: txHash,
	})

	// get the tx details
	txRes, err = c.grpc.GetTxResponse(txhash)
	blobId, blobHash, height = parse(txRes)

	// trigger ibc stateupdate - optional (?)
	// other ibc interactions would trigger this anyway. But until then, inclusion cannot be verified.
	// better to trigger a stateupdate now imo
	dymension.tx.ibc.client.updatestate(c.daConfig.clientID) // could import the go relayer and execute their funcs

	return ResultSubmitBatch{
		BaseResult:     BaseResult("success"),
		SubmitMetaData: ("interchain", height, blobId, blobHash, c.daConfig.clientID, )
	}
}

func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	panic("implement me")
}

func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	panic("implement me")
}
