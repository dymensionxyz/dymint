package avail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/types"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/author"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/chain"
	"github.com/centrifuge/go-substrate-rpc-client/v4/rpc/state"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	availtypes "github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
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
	maxBlobSize                     = 2097152 
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
	logger             types.Logger
	ctx                context.Context
	cancel             context.CancelFunc
	txInclusionTimeout time.Duration
	batchRetryDelay    time.Duration
	batchRetryAttempts uint
	synced             chan struct{}
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)


func WithClient(client SubstrateApiI) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).client = client
	}
}


func WithTxInclusionTimeout(timeout time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).txInclusionTimeout = timeout
	}
}


func WithBatchRetryDelay(delay time.Duration) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryDelay = delay
	}
}


func WithBatchRetryAttempts(attempts uint) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).batchRetryAttempts = attempts
	}
}


func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, _ store.KV, logger types.Logger, options ...da.Option) error {
	c.logger = logger
	c.synced = make(chan struct{}, 1)

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	}

	
	c.pubsubServer = pubsubServer
	c.txInclusionTimeout = defaultTxInculsionTimeout
	c.batchRetryDelay = defaultBatchRetryDelay
	c.batchRetryAttempts = defaultBatchRetryAttempts

	
	for _, apply := range options {
		apply(c)
	}

	
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

	types.RollappConsecutiveFailedDASubmission.Set(0)

	c.ctx, c.cancel = context.WithCancel(context.Background())
	return nil
}


func (c *DataAvailabilityLayerClient) Start() error {
	c.synced <- struct{}{}
	return nil
}


func (c *DataAvailabilityLayerClient) Stop() error {
	c.cancel()
	close(c.synced)
	return nil
}


func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
}


func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Avail
}


func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	
	blockHash, err := c.client.GetBlockHash(daMetaData.Height)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}
	block, err := c.client.GetBlock(blockHash)
	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}
	
	var batches []*types.Batch
	for _, ext := range block.Block.Extrinsics {
		
		if ext.Signature.AppID.Int64() == c.config.AppID &&
			ext.Method.CallIndex.SectionIndex == DataCallSectionIndex &&
			ext.Method.CallIndex.MethodIndex == DataCallMethodIndex {

			data := ext.Method.Args
			for 0 < len(data) {
				var pbBatch pb.Batch
				err := proto.Unmarshal(data, &pbBatch)
				if err != nil {
					c.logger.Error("unmarshal batch", "daHeight", daMetaData.Height, "error", err)
					continue
				}
				
				batch := &types.Batch{}
				err = batch.FromProto(&pbBatch)
				if err != nil {
					c.logger.Error("batch from proto", "daHeight", daMetaData.Height, "error", err)
					continue
				}
				
				batches = append(batches, batch)
				
				data = data[proto.Size(&pbBatch):]

			}
		}
	}

	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code: da.StatusSuccess,
		},
		CheckMetaData: &da.DACheckMetaData{
			Height: daMetaData.Height,
		},
		Batches: batches,
	}
}


func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}

	c.logger.Debug("Submitting to da batch with size", "size", len(blob))
	return c.submitBatchLoop(blob)
}



func (c *DataAvailabilityLayerClient) submitBatchLoop(dataBlob []byte) da.ResultSubmitBatch {
	for {
		select {
		case <-c.ctx.Done():
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "context done",
					Error:   c.ctx.Err(),
				},
			}
		default:
			var daBlockHeight uint64
			err := retry.Do(
				func() error {
					var err error
					daBlockHeight, err = c.broadcastTx(dataBlob)
					if err != nil {
						types.RollappConsecutiveFailedDASubmission.Inc()
						c.logger.Error("broadcasting batch", "error", err)
						if errors.Is(err, da.ErrTxBroadcastConfigError) {
							err = retry.Unrecoverable(err)
						}
						return err
					}
					return nil
				},
				retry.Context(c.ctx),
				retry.LastErrorOnly(true),
				retry.Delay(c.batchRetryDelay),
				retry.DelayType(retry.FixedDelay),
				retry.Attempts(c.batchRetryAttempts),
			)
			if err != nil {
				err = fmt.Errorf("broadcast data blob: %w", err)

				if !retry.IsRecoverable(err) {
					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: err.Error(),
							Error:   err,
						},
					}
				}

				c.logger.Error(err.Error())
				continue
			}
			types.RollappConsecutiveFailedDASubmission.Set(0)

			c.logger.Debug("Successfully submitted batch.")
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "success",
				},
				SubmitMetaData: &da.DASubmitMetaData{
					Client: da.Avail,
					Height: daBlockHeight,
				},
			}
		}
	}
}



func (c *DataAvailabilityLayerClient) broadcastTx(tx []byte) (uint64, error) {
	meta, err := c.client.GetMetadataLatest()
	if err != nil {
		return 0, fmt.Errorf("GetMetadataLatest: %w", err)
	}
	newCall, err := availtypes.NewCall(meta, DataCallSection+"."+DataCallMethod, availtypes.NewBytes(tx))
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}
	
	ext := availtypes.NewExtrinsic(newCall)
	genesisHash, err := c.client.GetBlockHash(0)
	if err != nil {
		return 0, fmt.Errorf("GetBlockHash: %w", err)
	}
	rv, err := c.client.GetRuntimeVersionLatest()
	if err != nil {
		return 0, fmt.Errorf("GetRuntimeVersionLatest: %w", err)
	}
	keyringPair, err := signature.KeyringPairFromSecret(c.config.Seed, keyringNetworkID)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}
	
	key, err := availtypes.CreateStorageKey(meta, "System", "Account", keyringPair.PublicKey)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}

	var accountInfo availtypes.AccountInfo
	ok, err := c.client.GetStorageLatest(key, &accountInfo)
	if err != nil || !ok {
		return 0, fmt.Errorf("GetStorageLatest: %w", err)
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

	
	err = ext.Sign(keyringPair, options)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", da.ErrTxBroadcastConfigError, err)
	}

	
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
					return 0, fmt.Errorf("getHeightFromHash: %w", err)
				}
				return blockHeight, nil
			} else if status.IsInBlock {
				c.logger.Debug(fmt.Sprintf("Batch included inside a block with hash %v, waiting for finalization.", status.AsInBlock.Hex()))
				inclusionTimer.Reset(c.txInclusionTimeout)
				continue
			} else {
				receivedStatus, err := status.MarshalJSON()
				if err != nil {
					return 0, fmt.Errorf("MarshalJSON of received status: %w", err)
				}
				c.logger.Debug("unsupported status, still waiting for inclusion", "status", string(receivedStatus))
				continue
			}
		case <-inclusionTimer.C:
			return 0, da.ErrTxBroadcastTimeout
		}
	}
}


func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "not implemented",
		},
	}
}


func (c *DataAvailabilityLayerClient) getHeightFromHash(hash availtypes.Hash) (uint64, error) {
	c.logger.Debug("Getting block height from hash", "hash", hash)
	header, err := c.client.GetHeader(hash)
	if err != nil {
		return 0, fmt.Errorf("cannot get block by hash:%w", err)
	}
	return uint64(header.Number), nil
}


func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	return maxBlobSize
}


func (c *DataAvailabilityLayerClient) GetSignerBalance() (da.Balance, error) {
	return da.Balance{}, nil
}
