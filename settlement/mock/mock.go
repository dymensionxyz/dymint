package mock

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const defaultBatchSize = 5
const kvStoreDBName = "settlement"

var settlementKVPrefix = []byte{0}
var latestHeightKey = []byte("latestHeight")
var slStateIndexKey = []byte("slStateIndex")

// SettlementLayerClient is intended only for usage in tests.
type SettlementLayerClient struct {
	logger       log.Logger
	settlementKV store.KVStore
	pubsub       *pubsub.Server
	latestHeight uint64
	slStateIndex uint64
	config       Config
	ctx          context.Context
	cancel       context.CancelFunc
}

// Config for the SettlementLayerClient mock
type Config struct {
	AutoUpdateBatches       bool          `json:"auto_update_batches"`
	AutoUpdateBatchInterval time.Duration `json:"auto_update_batch_interval"`
	BatchSize               uint64        `json:"batch_size"`
	LatestHeight            uint64        `json:"latest_height"`
	// The height at which the new batchSize started
	BatchOffsetHeight uint64 `json:"batch_offset_height"`
	DBPath            string `json:"db_path"`
	RootDir           string `json:"root_dir"`
	store             store.KVStore
}

var _ settlement.LayerClient = &SettlementLayerClient{}

// Init is called once. it initializes the struct members.
func (s *SettlementLayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger) error {
	c, err := s.getConfig(config)
	if err != nil {
		return err
	}
	s.config = *c
	s.pubsub = pubsub
	s.logger = logger
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.settlementKV = store.NewPrefixKV(s.config.store, settlementKVPrefix)
	b, err := s.settlementKV.Get(latestHeightKey)
	if err != nil {
		s.latestHeight = 0
	} else {
		s.latestHeight = binary.BigEndian.Uint64(b)
	}
	b, err = s.settlementKV.Get(slStateIndexKey)
	if err != nil {
		s.slStateIndex = 0
		s.latestHeight = 0
	} else {
		s.slStateIndex = binary.BigEndian.Uint64(b)
		// Get the latest height from the stateIndex
		var settlementBatch settlement.Batch
		b, err := s.settlementKV.Get(getKey(s.slStateIndex))
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &settlementBatch)
		if err != nil {
			return errors.New("error unmarshalling batch")
		}
		s.latestHeight = settlementBatch.EndHeight

	}
	return nil
}

func (s *SettlementLayerClient) decodeConfig(config []byte) (*Config, error) {
	var c Config
	err := json.Unmarshal(config, &c)
	return &c, err
}

func (s *SettlementLayerClient) getConfig(config []byte) (*Config, error) {
	var c *Config
	if len(config) > 0 {
		var err error
		c, err = s.decodeConfig(config)
		if err != nil {
			return nil, err
		}
		if c.BatchSize == 0 {
			c.BatchSize = defaultBatchSize
		}
		if c.RootDir != "" && c.DBPath != "" {
			c.store = store.NewDefaultKVStore(c.RootDir, c.DBPath, kvStoreDBName)
		} else {
			c.store = store.NewDefaultInMemoryKVStore()
		}
	} else {
		c = &Config{
			BatchSize: defaultBatchSize,
			store:     store.NewDefaultInMemoryKVStore(),
		}
	}
	return c, nil
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
	daResult := &da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			DAHeight: batch.EndHeight,
		},
	}
	s.SubmitBatch(&batch, da.Mock, daResult)
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
	// Save the batch to the next state index
	slStateIndex := atomic.LoadUint64(&s.slStateIndex)
	err = s.settlementKV.Set(getKey(slStateIndex+1), b)
	if err != nil {
		return err
	}
	// Save SL state index in memory and in store
	atomic.StoreUint64(&s.slStateIndex, slStateIndex+1)
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, slStateIndex+1)
	err = s.settlementKV.Set(slStateIndexKey, b)
	if err != nil {
		return err
	}
	// Save latest height in memory and in store
	atomic.StoreUint64(&s.latestHeight, batch.EndHeight)
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

func (s *SettlementLayerClient) convertBatchtoSettlementBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) *settlement.Batch {
	settlementBatch := &settlement.Batch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
		MetaData: &settlement.BatchMetaData{
			DA: &settlement.DAMetaData{
				Height: daResult.DAHeight,
				Client: daClient,
			},
		},
	}
	for _, block := range batch.Blocks {
		settlementBatch.AppHashes = append(settlementBatch.AppHashes, block.Header.AppHash)
	}
	return settlementBatch
}

// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (s *SettlementLayerClient) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) *settlement.ResultSubmitBatch {
	s.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	// validate batch
	err := s.validateBatch(batch)
	if err != nil {
		return &settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	// Build the result to save in the settlement layer.
	settlementBatch := s.convertBatchtoSettlementBatch(batch, daClient, daResult)
	// Save to the settlement layer
	err = s.saveBatch(settlementBatch)
	if err != nil {
		s.logger.Error("Error saving app hash to kv store", "error", err)
		return &settlement.ResultSubmitBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}
	}
	// Emit an event
	err = s.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewSettlementBatchAccepted{EndHeight: batch.EndHeight}, map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
	if err != nil {
		s.logger.Error("Error publishing event", "error", err)
	}
	return &settlement.ResultSubmitBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess, StateIndex: atomic.LoadUint64(&s.slStateIndex)},
	}
}

func (s *SettlementLayerClient) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	b, err := s.settlementKV.Get(getKey(slStateIndex))
	s.logger.Debug("Retrieving batch from settlement layer", "SL state index", slStateIndex)
	if err != nil {
		return nil, settlement.ErrBatchNotFound
	}
	var settlementBatch settlement.Batch
	err = json.Unmarshal(b, &settlementBatch)
	if err != nil {
		return nil, errors.New("error unmarshalling batch")
	}
	batchResult := settlement.ResultRetrieveBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess, StateIndex: slStateIndex},
		Batch:      &settlementBatch,
	}
	return &batchResult, nil
}

// RetrieveBatch Gets the batch which contains the given stateIndex. Empty stateIndex returns the latest batch.
func (s *SettlementLayerClient) RetrieveBatch(slStateIndex ...uint64) (*settlement.ResultRetrieveBatch, error) {
	var stateIndex uint64
	if len(slStateIndex) == 0 {
		stateIndex = atomic.LoadUint64(&s.slStateIndex)
		s.logger.Debug("Getting latest batch from settlement layer", "state index", stateIndex)
	} else if len(slStateIndex) == 1 {
		s.logger.Debug("Getting batch from settlement layer for SL state index", slStateIndex)
		stateIndex = slStateIndex[0]
	} else {
		return &settlement.ResultRetrieveBatch{}, errors.New("height len must be 1 or 0")
	}
	// Get the batch from the settlement layer
	batchResult, err := s.retrieveBatchAtStateIndex(stateIndex)
	if err != nil {
		return &settlement.ResultRetrieveBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}, err
	}
	return batchResult, nil

}

func getKey(key uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, key)
	return b
}
