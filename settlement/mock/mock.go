package mock

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/celestiaorg/optimint/log"
	"github.com/celestiaorg/optimint/settlement"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/testutil"
	"github.com/celestiaorg/optimint/types"
)

const defaultBatchSize = 5

// SettlementLayerClient is intended only for usage in tests.
type SettlementLayerClient struct {
	logger       log.Logger
	settlementKV store.KVStore
	latestHeight uint64
	config       Config
	ctx          context.Context
	cancel       context.CancelFunc
}

// Config for the SettlementLayerClient mock
type Config struct {
	AutoUpdateBatches       bool
	AutoUpdateBatchInterval time.Duration
	BatchSize               uint64
}

var _ settlement.LayerClient = &SettlementLayerClient{}

// Init is called once. it initializes the struct members.
func (s *SettlementLayerClient) Init(config []byte, settlementKV store.KVStore, logger log.Logger) error {
	s.logger = logger
	s.settlementKV = settlementKV
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
	s.SubmitBatch(&batch)
}

func (s *SettlementLayerClient) createBatch(startHeight uint64, endHeight uint64) types.Batch {
	s.logger.Debug("Creating batch for settlement layer", "start height", startHeight, "end height", endHeight)
	blocks := testutil.GenerateBlocks(int(endHeight - startHeight))
	batch := types.Batch{
		StartHeight: startHeight,
		EndHeight:   endHeight,
		Blocks:      blocks,
		// TODO(omritoptix): Change it to be received as func arg
		DAPath: fmt.Sprint(endHeight),
	}
	return batch
}

// saveBatch saves the data to the kvstore
func (s *SettlementLayerClient) saveBatch(resultRetrieveBatch *settlement.ResultRetrieveBatch) error {
	s.logger.Debug("Saving batch to settlement layer", "start height",
		resultRetrieveBatch.StartHeight, "end height", resultRetrieveBatch.EndHeight)
	b, err := json.Marshal(resultRetrieveBatch)
	if err != nil {
		return err
	}
	err = s.settlementKV.Set(getKey(resultRetrieveBatch.EndHeight), b)
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

// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (s *SettlementLayerClient) SubmitBatch(batch *types.Batch) settlement.ResultSubmitBatch {
	s.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	// validate batch
	err := s.validateBatch(batch)
	if err != nil {
		return settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	// Build the result to save in the settlement layer.
	batchResult := &settlement.ResultRetrieveBatch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
	}
	for _, block := range batch.Blocks {
		batchResult.AppHashes = append(batchResult.AppHashes, block.Header.AppHash)
	}
	// Save to the settlement layer
	err = s.saveBatch(batchResult)
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
	if err != nil {
		return nil, errors.New("error getting batch from kv store for height " + fmt.Sprint(endHeight))
	}
	var batchResult settlement.ResultRetrieveBatch
	err = json.Unmarshal(b, &batchResult)
	if err != nil {
		return nil, errors.New("error unmarshalling batch")
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
			}, errors.New("error getting latest batch")
		}
		return *batchResult, nil

	} else if len(height) == 1 {
		s.logger.Debug("Getting batch from settlement layer for height", height)
		height := height[0]
		endHeight := uint64(float64(height/s.config.BatchSize)) * (s.config.BatchSize + 1)
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
