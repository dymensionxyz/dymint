package mock

import (
	"crypto/sha1" //#nosec
	"encoding/binary"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// DataAvailabilityLayerClient is intended only for usage in tests.
// It does actually ensures DA - it stores data in-memory.
type DataAvailabilityLayerClient struct {
	logger   log.Logger
	dalcKV   store.KVStore
	daHeight uint64
	config   config
}

const defaultBlockTime = 3 * time.Second

type config struct {
	BlockTime time.Duration
}

var _ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
var _ da.BatchRetriever = &DataAvailabilityLayerClient{}

// Init is called once to allow DA client to read configuration and initialize resources.
func (m *DataAvailabilityLayerClient) Init(config []byte, _ *pubsub.Server, dalcKV store.KVStore, logger log.Logger, options ...da.Option) error {
	m.logger = logger
	m.dalcKV = dalcKV
	m.daHeight = 1
	if len(config) > 0 {
		var err error
		m.config.BlockTime, err = time.ParseDuration(string(config))
		if err != nil {
			return err
		}
	} else {
		m.config.BlockTime = defaultBlockTime
	}
	return nil
}

// Start implements DataAvailabilityLayerClient interface.
func (m *DataAvailabilityLayerClient) Start() error {
	m.logger.Debug("Mock Data Availability Layer Client starting")
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
	return nil
}

// GetClientType returns client type.
func (m *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Mock
}

// SubmitBatch submits the passed in batch to the DA layer.
// This should create a transaction which (potentially)
// triggers a state transition in the DA layer.
func (m *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	daHeight := atomic.LoadUint64(&m.daHeight)
	m.logger.Debug("Submitting batch to DA layer", "start height", batch.StartHeight, "end height", batch.EndHeight, "da height", daHeight)

	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}
	hash := sha1.Sum(uint64ToBinary(batch.EndHeight)) //#nosec
	err = m.dalcKV.Set(getKey(daHeight, batch.StartHeight), hash[:])
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}
	err = m.dalcKV.Set(hash[:], blob)
	if err != nil {
		return da.ResultSubmitBatch{BaseResult: da.BaseResult{Code: da.StatusError, Message: err.Error()}}
	}

	atomic.StoreUint64(&m.daHeight, daHeight+1)

	return da.ResultSubmitBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "OK",
			MetaData: &da.DAMetaData{
				Height: daHeight,
			},
		},
	}
}

// CheckBatchAvailability queries DA layer to check data availability of block corresponding to given header.
func (m *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DAMetaData) da.ResultCheckBatch {

	batchesRes := m.RetrieveBatches(daMetaData)
	return da.ResultCheckBatch{BaseDACheckResult: da.BaseDACheckResult{Code: batchesRes.Code}, DataAvailable: len(batchesRes.Batches) > 0}
}

// RetrieveBatches returns block at given height from data availability layer.
func (m *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DAMetaData) da.ResultRetrieveBatch {
	if daMetaData.Height >= atomic.LoadUint64(&m.daHeight) {
		return da.ResultRetrieveBatch{BaseDACheckResult: da.BaseDACheckResult{Code: da.StatusError, Message: "batch not found"}}
	}

	iter := m.dalcKV.PrefixIterator(uint64ToBinary(daMetaData.Height))
	defer iter.Discard()

	var batches []*types.Batch
	for iter.Valid() {
		hash := iter.Value()

		blob, err := m.dalcKV.Get(hash)
		if err != nil {
			return da.ResultRetrieveBatch{BaseDACheckResult: da.BaseDACheckResult{Code: da.StatusError, Message: err.Error()}}
		}

		batch := &types.Batch{}
		err = batch.UnmarshalBinary(blob)
		if err != nil {
			return da.ResultRetrieveBatch{BaseDACheckResult: da.BaseDACheckResult{Code: da.StatusError, Message: err.Error()}}
		}
		batches = append(batches, batch)

		iter.Next()
	}
	DACheckMetaData := &da.DACheckMetaData{Height: daMetaData.Height}
	return da.ResultRetrieveBatch{BaseDACheckResult: da.BaseDACheckResult{Code: da.StatusSuccess, DACheckMetaData: DACheckMetaData}, Batches: batches}
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
	atomic.AddUint64(&m.daHeight, blockStep)
}
