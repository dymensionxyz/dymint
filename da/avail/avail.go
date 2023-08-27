package avail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/log"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/chain"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/state"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	availtypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	keyringNetworkID          uint8 = 42
	defaultTxInculsionTimeout       = 100 * time.Second
	defaultBatchRetryDelay          = 10 * time.Second
	defaultBatchRetryAttempts       = 10
	DataCallSection                 = "DataAvailability"
	DataCallMethod                  = "submit_data"
	DataCallSectionIndex            = 29
	DataCallMethodIndex             = 1
)

type SubstrateApiI interface {
	chain.Chain
	state.State
	author.Author
}

type SubstrateApi struct {
	chain.Chain
	state.State
	author.Author
}

type Config struct {
	Seed   string `json:"seed"`
	ApiURL string `json:"api_url"`
	AppID  int64  `json:"app_id"`
	Tip    uint64 `json:"tip"`
}

type DataAvailabilityLayerClient struct {
	client             SubstrateApiI
	pubsubServer       *pubsub.Server
	config             Config
	logger             log.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	txInclusionTimeout time.Duration
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// WithClient is an option which sets the client.
func WithClient(client SubstrateApiI) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).client = client
	}
}

// WithTxInclusionTimeout is an option which sets the timeout for waiting for transaction inclusion.
func WithTxInclusionTimeout(timeout time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).txInclusionTimeout = timeout
	}
}

// WithBatchRetryDelay is an option which sets the delay between batch retries.
func WithBatchRetryDelay(delay time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryDelay = delay
	}
}

// WithBatchRetryAttempts is an option which sets the number of batch retries.
func WithBatchRetryAttempts(attempts uint) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryAttempts = attempts
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KVStore, logger log.Logger, options ...da.Option) error {
	c.logger = logger

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	}

	// Set defaults
	c.pubsubServer = pubsubServer
	c.txInclusionTimeout = defaultTxInculsionTimeout
	c.batchRetryDelay = defaultBatchRetryDelay
	c.batchRetryAttempts = defaultBatchRetryAttempts

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	// If client wasn't set, create a new one
	if c.client == nil {
		substrateApiClient, err := gsrpc.NewSubstrateAPI(c.config.ApiURL)
		if err != nil {
			return err
		}
		c.client = SubstrateApi{
			Chain:  substrateApiClient.RPC.Chain,
			State:  substrateApiClient.RPC.State,
			Author: substrateApiClient.RPC.Author,
		}
	}

	c.ctx, c.cancel = context.WithCancel(context.Background())

	return nil
}

// Start starts DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Start() error {
	return nil
}

// Stop stops DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.cancel()
	return nil
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Avail
}

// RetrieveBatch retrieves batch from DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) RetrieveBatches(dataLayerHeight uint64) da.ResultRetrieveBatch {
	blockHash, err := c.client.GetBlockHash(dataLayerHeight)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}
	block, err := c.client.GetBlock(blockHash)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}

	}
	// Convert the data returned to batches
	var batches []*types.Batch
	for _, ext := range block.Block.Extrinsics {
		// these values below are specific indexes only for data submission, differs with each extrinsic
		if ext.Signature.AppID.Int64() == c.config.AppID &&
			ext.Method.CallIndex.SectionIndex == DataCallSectionIndex &&
			ext.Method.CallIndex.MethodIndex == DataCallMethodIndex {
			data := ext.Method.Args
			for len(data) > 0 {
				var pbBatch pb.Batch
				// Attempt to unmarshal the data.
				err := proto.Unmarshal(data, &pbBatch)
				if err != nil {
					c.logger.Error("failed to unmarshal batch", "daHeight", dataLayerHeight, "error", err)
					continue
				}
				// Convert the proto batch to a batch
				batch := &types.Batch{}
				err = batch.FromProto(&pbBatch)
				if err != nil {
					c.logger.Error("failed to convert batch", "daHeight", dataLayerHeight, "error", err)
					continue
				}
				// Add the batch to the list
				batches = append(batches, batch)
				// Remove the bytes we just decoded.
				data = data[proto.Size(&pbBatch):]

			}
		}
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:     da.StatusSuccess,
			DAHeight: dataLayerHeight,
		},
		Batches: batches,
	}
}

// SubmitBatch submits batch to DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
			},
		}
	}

	c.logger.Debug("Submitting to da batch with size", "size", len(blob))
	return c.submitBatchLoop(blob)

}

// submitBatchLoop tries submitting the batch. In case we get a configuration error we would like to stop trying,
// otherwise, for network error we keep trying indefinitely.
func (c *DataAvailabilityLayerClient) submitBatchLoop(dataBlob []byte) da.ResultSubmitBatch {
	for {
		select {
		case <-c.ctx.Done():
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "context done",
				},
			}
		default:
			var daBlockHeight uint64
			err := retry.Do(func() error {
				var err error
				daBlockHeight, err = c.broadcastTx(dataBlob)
				if err != nil {
					c.logger.Error("Error broadcasting batch", "error", err)
					if errors.Is(err, da.ErrTxBroadcastConfigError) {
						err = retry.Unrecoverable(err)
					}
					return err
				}
				return nil
			}, retry.Context(c.ctx), retry.LastErrorOnly(true), retry.Delay(c.batchRetryDelay),
				retry.DelayType(retry.FixedDelay), retry.Attempts(c.batchRetryAttempts))
			if err != nil {
				if !retry.IsRecoverable(err) {
					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: err.Error(),
						},
					}
				} else {
					c.logger.Error("Error broadcasting batch. Emitting DA unhealthy event and Trying again.", "error", err)
					res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, false, err)
					if err != nil {
						return res
					}
					continue
				}
			}

			c.logger.Debug("Successfully submitted DA batch")
			res, err := da.SubmitBatchHealthEventHelper(c.pubsubServer, c.ctx, true, nil)
			if err != nil {
				return res
			}
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:     da.StatusSuccess,
					Message:  "success",
					DAHeight: daBlockHeight,
				},
			}

		}
	}

}

// broadcastTx broadcasts the transaction to the network and in case of success
// returns the block height the batch was included in.
func (c *DataAvailabilityLayerClient) broadcastTx(tx []byte) (uint64, error) {
	meta, err := c.client.GetMetadataLatest()
	if err != nil {
		return 0, fmt.Errorf("%s: %s", "failed to GetMetadataLatest", err)
	}
	newCall, err := availtypes.NewCall(meta, DataCallSection+"."+DataCallMethod, availtypes.NewBytes(tx))
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}
	// Create the extrinsic
	ext := availtypes.NewExtrinsic(newCall)
	genesisHash, err := c.client.GetBlockHash(0)
	if err != nil {
		return 0, fmt.Errorf("%s: %s", "failed to GetBlockHash", err)
	}
	rv, err := c.client.GetRuntimeVersionLatest()
	if err != nil {
		return 0, fmt.Errorf("%s: %s", "failed to GetRuntimeVersionLatest", err)
	}
	keyringPair, err := signature.KeyringPairFromSecret(c.config.Seed, keyringNetworkID)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}
	// Get the account info for the nonce
	key, err := availtypes.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}

	var accountInfo availtypes.AccountInfo
	ok, err := c.client.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return 0, fmt.Errorf("%s: %s", "failed to GetStorageLatest", err)
	}

	nonce := uint32(accountInfo.Nonce)
	options := availtypes.SignatureOptions{
		BlockHash:          genesisHash,
		Era:                availtypes.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        genesisHash,
		Nonce:              availtypes.NewUCompactFromUInt(uint64(nonce)),
		SpecVersion:        rv.SpecVersion,
		Tip:                availtypes.NewUCompactFromUInt(c.config.Tip),
		TransactionVersion: rv.TransactionVersion,
		AppID:              availtypes.NewUCompactFromUInt(uint64(c.config.AppID)),
	}

	// Sign the transaction using Alice's default account
	err = ext.Sign(keyringPair, options)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}

	// Send the extrinsic
	sub, err := c.client.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastNetworkError, err)
	}

	c.logger.Info("Submitted batch to avail. Waiting for inclusion event")

	defer sub.Unsubscribe()

	inclusionTimer := time.NewTimer(c.txInclusionTimeout)
	defer inclusionTimer.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return 0, c.ctx.Err()
		case err := <-sub.Err():
			return 0, err
		case status := <-sub.Chan():
			if status.IsFinalized {
				c.logger.Debug("Batch finalized inside block")
				hash := status.AsFinalized
				blockHeight, err := c.getHeightFromHash(hash)
				if err != nil {
					return 0, fmt.Errorf("%s: %s", "failed to getHeightFromHash", err)
				}
				return blockHeight, nil
			} else if status.IsInBlock {
				c.logger.Debug(fmt.Sprintf("Batch included inside a block with hash %v, waiting for finalization.", status.AsInBlock.Hex()))
				inclusionTimer.Reset(c.txInclusionTimeout)
				continue
			} else {
				recievedStatus, err := status.MarshalJSON()
				if err != nil {
					return 0, fmt.Errorf("%s: %s", "failed to MarshalJSON of received status", err)
				}
				c.logger.Debug("unsupported status, still waiting for inclusion", "status", string(recievedStatus))
				continue
			}
		case <-inclusionTimer.C:
			return 0, da.ErrTxBroadcastTimeout
		}
	}
}

// CheckBatchAvailability checks batch availability in DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(dataLayerHeight uint64) da.ResultCheckBatch {
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusError,
			Message: "not implemented",
		},
	}
}

// getHeightFromHash returns the block height from the block hash
func (c *DataAvailabilityLayerClient) getHeightFromHash(hash availtypes.Hash) (uint64, error) {
	c.logger.Debug("Getting block height from hash", "hash", hash)
	header, err := c.client.GetHeader(hash)
	if err != nil {
		return 0, fmt.Errorf("cannot get block by hash:%w", err)
	}
	return uint64(header.Number), nil
}
