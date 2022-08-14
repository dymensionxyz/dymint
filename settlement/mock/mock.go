package mock

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/celestiaorg/optimint/da"
	"github.com/celestiaorg/optimint/log"
	"github.com/celestiaorg/optimint/settlement"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/testutil"
	"github.com/celestiaorg/optimint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const defaultBatchSize = 5

var settlementKVPrefix = []byte{0}

// SettlementLayerClient is intended only for usage in tests.
type SettlementLayerClient struct {
	logger            log.Logger
	settlementKV      store.KVStore
	pubsub            *pubsub.Server
	latestHeight      uint64
	batchOffsetHeight uint64
	config            Config
	ctx               context.Context
	cancel            context.CancelFunc
}

// Config for the SettlementLayerClient mock
type Config struct {
	AutoUpdateBatches       bool          `json:"auto_update_batches"`
	AutoUpdateBatchInterval time.Duration `json:"auto_update_batch_interval"`
	BatchSize               uint64        `json:"batch_size"`
	LatestHeight            uint64        `json:"latest_height"`
	// The height at which the new batchSize started
	BatchOffsetHeight uint64 `json:"batch_offset_height"`
}

var _ settlement.LayerClient = &SettlementLayerClient{}

// Init is called once. it initializes the struct members.
func (s *SettlementLayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger) error {
	s.pubsub = pubsub
	s.logger = logger
	baseKV := store.NewDefaultInMemoryKVStore()
	s.settlementKV = store.NewPrefixKV(baseKV, settlementKVPrefix)
	s.latestHeight = 0
	s.ctx, s.cancel = context.WithCancel(context.Background())
	if len(config) > 0 {
		var err error
		s.config, err = s.decodeConfig(config)
		if err != nil {
			return err
		}
		if s.config.BatchSize == 0 {
			s.config.BatchSize = defaultBatchSize
		}
		if s.config.LatestHeight > 0 {
			s.latestHeight = s.config.LatestHeight
		}
		s.batchOffsetHeight = s.config.BatchOffsetHeight
	}
	return nil
}

func (s *SettlementLayerClient) decodeConfig(config []byte) (Config, error) {
	var c Config
	err := json.Unmarshal(config, &c)
	return c, err
}

// Start is called once, after init. If configured so, it will start producing batches every interval.
func (s *SettlementLayerClient) Start() error {
	s.logger.Debug("Mock settlement Layer Client starting")
	if s.config.AutoUpdateBatches {
		go func() {
			timer := time.NewTimer(s.config.AutoUpdateBatchInterval)
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-timer.C:
					s.updateSettlementWithBatch()
				}
			}
		}()
	}
	return nil
}

// Stop is called once, after Start. it cancels the auto batches created, if such was started.
func (s *SettlementLayerClient) Stop() error {
	s.logger.Debug("Mock settlement Layer Client stopping")
	s.cancel()
	return nil
}

func (s *SettlementLayerClient) updateSettlementWithBatch() {
	s.logger.Debug("Mock settlement Layer Client updating with batch")
	batch := s.createBatch(s.latestHeight+1, s.latestHeight+1+s.config.BatchSize)
	batchMetaData := &settlement.BatchMetaData{
		DA: &settlement.DAMetaData{
			Height: batch.EndHeight,
			Path:   strconv.FormatUint(batch.EndHeight, 10),
			Client: da.Celestia,
		},
	}
	s.SubmitBatch(&batch, batchMetaData)
}

func (s *SettlementLayerClient) createBatch(startHeight uint64, endHeight uint64) types.Batch {
	s.logger.Debug("Creating batch for settlement layer", "start height", startHeight, "end height", endHeight)
	blocks := testutil.GenerateBlocks(startHeight, endHeight-startHeight)
	batch := types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      blocks,
	}
	return batch
}

// saveBatch saves the data to the kvstore
func (s *SettlementLayerClient) saveBatch(batch *settlement.Batch) error {
	s.logger.Debug("Saving batch to settlement layer", "start height",
		batch.StartHeight, "end height", batch.EndHeight)
	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	err = s.settlementKV.Set(getKey(batch.EndHeight), b)
	if err != nil {
		return err
	}
	return nil
}

func (s *SettlementLayerClient) validateBatch(batch *types.Batch) error {
	if batch.StartHeight != s.latestHeight+1 {
		return errors.New("batch start height must be last height + 1")
	}
	if batch.EndHeight < batch.StartHeight {
		return errors.New("batch end height must be greater or equal to start height")
	}
	return nil
}

func (s *SettlementLayerClient) convertBatchtoSettlementBatch(batch *types.Batch, metaData *settlement.BatchMetaData) *settlement.Batch {
	settlementBatch := &settlement.Batch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
		MetaData:    metaData,
	}
	for _, block := range batch.Blocks {
		settlementBatch.AppHashes = append(settlementBatch.AppHashes, block.Header.AppHash)
	}
	return settlementBatch
}

// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (s *SettlementLayerClient) SubmitBatch(batch *types.Batch, metaData *settlement.BatchMetaData) settlement.ResultSubmitBatch {
	s.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	// validate batch
	err := s.validateBatch(batch)
	if err != nil {
		return settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	// Build the result to save in the settlement layer.
	settlementBatch := s.convertBatchtoSettlementBatch(batch, metaData)
	// Save to the settlement layer
	err = s.saveBatch(settlementBatch)
	if err != nil {
		s.logger.Error("Error saving app hash to kv store", "error", err)
		return settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	s.latestHeight = batch.EndHeight
	return settlement.ResultSubmitBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess},
	}
}

func (s *SettlementLayerClient) retrieveBatchAtEndHeight(endHeight uint64) (*settlement.ResultRetrieveBatch, error) {
	b, err := s.settlementKV.Get(getKey(endHeight))
	s.logger.Debug("Retrieving batch from settlement layer", "end height", endHeight)
	if err != nil {
		return nil, settlement.ErrBatchNotFound
	}
	var settlementBatch settlement.Batch
	err = json.Unmarshal(b, &settlementBatch)
	if err != nil {
		return nil, errors.New("error unmarshalling batch")
	}
	batchResult := settlement.ResultRetrieveBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess},
		Batch:      &settlementBatch,
	}
	return &batchResult, nil
}

// RetrieveBatch Gets the batch which contains the given height. Empty height returns the latest batch.
func (s *SettlementLayerClient) RetrieveBatch(height ...uint64) (settlement.ResultRetrieveBatch, error) {
	if len(height) == 0 {
		s.logger.Debug("Getting latest batch from settlement layer", "latest height", s.latestHeight)
		batchResult, err := s.retrieveBatchAtEndHeight(s.latestHeight)
		if err != nil {
			return settlement.ResultRetrieveBatch{
				BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
			}, err
		}
		return *batchResult, nil

	} else if len(height) == 1 {
		s.logger.Debug("Getting batch from settlement layer for height", height)
		height := height[0]
		startHeight := (height-s.config.BatchOffsetHeight)/s.config.BatchSize*s.config.BatchSize + s.config.BatchOffsetHeight
		endHeight := startHeight + (s.config.BatchSize - 1)
		resultRetrieveBatch, err := s.retrieveBatchAtEndHeight(endHeight)
		if err != nil {
			return settlement.ResultRetrieveBatch{
				BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
			}, errors.New("error getting batch from kv store at height " + fmt.Sprint(endHeight))
		}
		return *resultRetrieveBatch, nil
	} else {
		return settlement.ResultRetrieveBatch{}, errors.New("height len must be 1 or 0")
	}

}

func getKey(endHeight uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, endHeight)
	return b
}
