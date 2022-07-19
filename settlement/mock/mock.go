package mock

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"

	"github.com/celestiaorg/optimint/log"
	"github.com/celestiaorg/optimint/settlement"
	"github.com/celestiaorg/optimint/store"
	"github.com/celestiaorg/optimint/types"
)

// SettlementLayerClient is intended only for usage in tests.
type SettlementLayerClient struct {
	logger       log.Logger
	settlementKV store.KVStore
	latestHeight uint64
}

var _ settlement.SettlementLayerClient = &SettlementLayerClient{}

const LatestBatchHeight = -1

func (s *SettlementLayerClient) Init(config []byte, settlementKV store.KVStore, logger log.Logger) error {
	s.logger = logger
	s.settlementKV = settlementKV
	s.latestHeight = 0
	return nil
}

// SubmitBatch submits the batch to the settlement layer.
// This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (s *SettlementLayerClient) SubmitBatch(batch *types.Batch) settlement.ResultSubmitBatch {
	s.logger.Debug("Submitting batch to settlement layer!", "start height", batch.StartHeight, "end height", batch.EndHeight)
	// Build the result to save in the settlement layer.
	batchResult := settlement.ResultGetLatestBatch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
	}
	for _, block := range batch.Blocks {
		batchResult.AppHashes = append(batchResult.AppHashes, block.Header.AppHash)
	}

	// Save to the settlement layer
	var b bytes.Buffer // buffer to store the result
	enc := gob.NewEncoder(&b)
	enc.Encode(batchResult)
	err := s.settlementKV.Set(getKey(batch.EndHeight), b.Bytes())
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

// GetLatestBatch gets the latest batch height from the settlement layer.
func (s *SettlementLayerClient) GetLatestBatch() settlement.ResultGetLatestBatch {
	s.logger.Debug("Getting latest batch from settlement layer!", "latest height", s.latestHeight)
	if s.latestHeight == 0 {
		return settlement.ResultGetLatestBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: "No batch found"},
		}
	}
	// Get the latest batch from the settlement layer.
	blob, err := s.settlementKV.Get(getKey(s.latestHeight))
	if err != nil {
		s.logger.Error("Error getting latest batch from kv store", "error", err)
		return settlement.ResultGetLatestBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	var batchResult settlement.ResultGetLatestBatch
	dec := gob.NewDecoder(bytes.NewReader(blob))
	dec.Decode(&batchResult)
	return batchResult
}

func getKey(endHeight uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, endHeight)
	return b
}
