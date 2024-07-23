package local

import (
	"crypto/sha1" //#nosec
	"encoding/binary"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// DataAvailabilityLayerClient is intended only for usage in tests.
// It does actually ensures DA - it stores data in-memory.
type DataAvailabilityLayerClient struct {
	logger   types.Logger
	dalcKV   store.KV
	daHeight atomic.Uint64
	config   config
	synced   chan struct{}
}

const defaultBlockTime = 3 * time.Second

type config struct {
	BlockTime time.Duration
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// Init is called once to allow DA client to read configuration and initialize resources.
func (m *DataAvailabilityLayerClient) Init(config []byte, _ *pubsub.Server, dalcKV store.KV, logger types.Logger, options ...da.Option) error {
	m.logger = logger
	m.dalcKV = dalcKV
	m.daHeight.Store(1)
	if len(config) > 0 {
		var err error
		m.config.BlockTime, err = time.ParseDuration(string(config))
		if err != nil {
			return err
		}
	} else {
		m.config.BlockTime = defaultBlockTime
	}
	m.synced = make(chan struct{}, 1)
	return nil
}

// Start implements DataAvailabilityLayerClient interface.
func (m *DataAvailabilityLayerClient) Start() error {
	m.logger.Debug("Mock Data Availability Layer Client starting")
	m.synced <- struct{}{}
	go func() {
		for {
			time.Sleep(m.config.BlockTime)
			m.updateDAHeight()
		}
	}()

	return nil
}

// Stop implements DataAvailabilityLayerClient interface.
func (m *DataAvailabilityLayerClient) Stop() error {
	m.logger.Debug("Mock Data Availability Layer Client stopped")
	close(m.synced)
	return nil
}

// Synced returns channel for on start event
func (m *DataAvailabilityLayerClient) Synced() <-chan struct{} {
	return m.synced
}

// GetClientType returns client type.
func (m *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Mock
}

// SubmitBatch submits the passed in batch to the DA layer.
// This should create a transaction which (potentially)
// triggers a state transition in the DA layer.
func (m *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	daHeight := m.daHeight.Load()

	m.logger.Debug("Submitting batch to DA layer", "start height", batch.StartHeight(), "end height", batch.EndHeight(), "da height", daHeight)

	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
	}
	hash := sha1.Sum(uint64ToBinary(batch.EndHeight())) //#nosec
	err = m.dalcKV.Set(getKey(daHeight, batch.StartHeight()), hash[:])
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
	}
	err = m.dalcKV.Set(hash[:], blob)
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
	}

	m.daHeight.Store(daHeight + 1) // guaranteed no ABA problem as submit batch is only called when the object is locked

	return da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "OK",
		},
		SubmitMetaData: &da.DASubmitMetaData{
			Height: daHeight,
			Client: da.Mock,
		},
	}
}

// CheckBatchAvailability queries DA layer to check data availability of block corresponding to given header.
func (m *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	batchesRes := m.RetrieveBatches(daMetaData)
	return da.ResultCheckBatch{BaseResult: da.BaseResult{Code: batchesRes.Code, Message: batchesRes.Message, Error: batchesRes.Error}, CheckMetaData: batchesRes.CheckMetaData}
}

// RetrieveBatches returns block at given height from data availability layer.
func (m *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	if daMetaData.Height >= m.daHeight.Load() {
		return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: "batch not found", Error: da.ErrBlobNotFound}}
	}

	iter := m.dalcKV.PrefixIterator(uint64ToBinary(daMetaData.Height))
	defer iter.Discard()

	var batches []*types.Batch
	for iter.Valid() {
		hash := iter.Value()

		blob, err := m.dalcKV.Get(hash)
		if err != nil {
			return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
		}

		batch := &types.Batch{}
		err = batch.UnmarshalBinary(blob)
		if err != nil {
			return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error(), Error: err}}
		}
		batches = append(batches, batch)

		iter.Next()
	}
	DACheckMetaData := &da.DACheckMetaData{Height: daMetaData.Height}
	return da.ResultRetrieveBatch{BaseResult: da.BaseResult{Code: da.StatusSuccess}, CheckMetaData: DACheckMetaData, Batches: batches}
}

func uint64ToBinary(daHeight uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, daHeight)
	return b
}

func getKey(daHeight uint64, height uint64) []byte {
	b := make([]byte, 16)
	binary.BigEndian.PutUint64(b, daHeight)
	binary.BigEndian.PutUint64(b[8:], height)
	return b
}

func (m *DataAvailabilityLayerClient) updateDAHeight() {
	blockStep := rand.Uint64()%10 + 1 //#nosec
	m.daHeight.Add(blockStep)
}
