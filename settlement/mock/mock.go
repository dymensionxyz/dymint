package mock

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"sync/atomic"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/pubsub"
)

const kvStoreDBName = "settlement"

var settlementKVPrefix = []byte{0}
var slStateIndexKey = []byte("slStateIndex")

// SettlementLayerClient is an extension of the base settlement layer client
// for usage in tests and local development.
type SettlementLayerClient struct {
	*settlement.BaseLayerClient
}

var _ settlement.LayerClient = (*SettlementLayerClient)(nil)

// Init initializes the mock layer client.
func (m *SettlementLayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger, options ...settlement.Option) error {
	HubClientMock, err := newHubClient(config, pubsub, logger)
	if err != nil {
		return err
	}
	baseOptions := []settlement.Option{
		settlement.WithHubClient(HubClientMock),
	}
	if options == nil {
		options = baseOptions
	} else {
		options = append(baseOptions, options...)
	}
	m.BaseLayerClient = &settlement.BaseLayerClient{}
	err = m.BaseLayerClient.Init(config, pubsub, logger, options...)
	if err != nil {
		return err
	}
	return nil
}

// Config for the HubClient
type Config struct {
	*settlement.Config
	DBPath  string `json:"db_path"`
	RootDir string `json:"root_dir"`
	store   store.KVStore
}

// HubClient implements The HubClient interface
type HubClient struct {
	config       *Config
	slStateIndex uint64
	logger       log.Logger
	pubsub       *pubsub.Server
	latestHeight uint64
	settlementKV store.KVStore
}

var _ settlement.HubClient = &HubClient{}

// PostBatchResp is the response from saving the batch
type PostBatchResp struct {
	err error
}

// GetTxHash returns the tx hash
func (s PostBatchResp) GetTxHash() string {
	if s.err != nil {
		return ""
	}
	return "mock-hash"
}

// GetCode returns the code
func (s PostBatchResp) GetCode() uint32 {
	if s.err != nil {
		return 1
	}
	return 0
}

var _ settlement.PostBatchResp = PostBatchResp{}

func newHubClient(config []byte, pubsub *pubsub.Server, logger log.Logger) (*HubClient, error) {
	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	conf, err := getConfig(config)
	if err != nil {
		return nil, err
	}
	settlementKV := store.NewPrefixKV(conf.store, settlementKVPrefix)
	b, err := settlementKV.Get(slStateIndexKey)
	if err == nil {
		slStateIndex = binary.BigEndian.Uint64(b)
		// Get the latest height from the stateIndex
		var settlementBatch rollapptypes.MsgUpdateState
		b, err := settlementKV.Get(getKey(slStateIndex))
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, &settlementBatch)
		if err != nil {
			return nil, errors.New("error unmarshalling batch")
		}
		latestHeight = settlementBatch.StartHeight + settlementBatch.NumBlocks - 1
	}
	return &HubClient{
		config:       conf,
		logger:       logger,
		pubsub:       pubsub,
		latestHeight: latestHeight,
		slStateIndex: slStateIndex,
		settlementKV: settlementKV,
	}, nil
}

func getConfig(config []byte) (*Config, error) {
	var conf *Config
	if len(config) > 0 {
		var err error
		conf, err = decodeConfig(config)
		if err != nil {
			return nil, err
		}
		if conf.RootDir != "" && conf.DBPath != "" {
			conf.store = store.NewDefaultKVStore(conf.RootDir, conf.DBPath, kvStoreDBName)
		} else {
			conf.store = store.NewDefaultInMemoryKVStore()
		}
	} else {
		conf = &Config{
			store: store.NewDefaultInMemoryKVStore(),
		}
	}
	return conf, nil
}

func decodeConfig(config []byte) (*Config, error) {
	var conf Config
	err := json.Unmarshal(config, &conf)
	return &conf, err
}

// Start starts the mock client
func (c *HubClient) Start() error {
	return nil
}

// Stop stops the mock client
func (c *HubClient) Stop() error {
	return nil
}

// PostBatch saves the batch to the kv store
func (c *HubClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) (settlement.PostBatchResp, error) {
	settlementBatch := c.convertBatchtoSettlementBatch(batch, daClient, daResult)
	err := c.saveBatch(settlementBatch)
	if err != nil {
		return PostBatchResp{err}, err
	}
	err = c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewSettlementBatchAccepted{EndHeight: settlementBatch.EndHeight}, map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
	if err != nil {
		c.logger.Error("error publishing event", "error", err)
	}
	return PostBatchResp{nil}, nil
}

// GetLatestBatch returns the latest batch from the kv store
func (c *HubClient) GetLatestBatch(rollappID string) (*settlement.ResultRetrieveBatch, error) {
	batchResult, err := c.GetBatchAtIndex(rollappID, atomic.LoadUint64(&c.slStateIndex))
	if err != nil {
		return nil, err
	}
	return batchResult, nil
}

// GetBatchAtIndex returns the batch at the given index
func (c *HubClient) GetBatchAtIndex(rollappID string, index uint64) (*settlement.ResultRetrieveBatch, error) {
	batchResult, err := c.retrieveBatchAtStateIndex(index)
	if err != nil {
		return &settlement.ResultRetrieveBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}, err
	}
	return batchResult, nil
}

// GetSequencers returns a list of sequencers. Currently only returns a single sequencer
func (c *HubClient) GetSequencers(rollappID string) ([]*types.Sequencer, error) {
	return []*types.Sequencer{
		{
			PublicKey: secp256k1.GenPrivKey().PubKey(),
			Status:    types.Proposer,
		},
	}, nil
}

func (c *HubClient) saveBatch(batch *settlement.Batch) error {
	c.logger.Debug("Saving batch to settlement layer", "start height",
		batch.StartHeight, "end height", batch.EndHeight)
	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	// Save the batch to the next state index
	slStateIndex := atomic.LoadUint64(&c.slStateIndex)
	err = c.settlementKV.Set(getKey(slStateIndex+1), b)
	if err != nil {
		return err
	}
	// Save SL state index in memory and in store
	atomic.StoreUint64(&c.slStateIndex, slStateIndex+1)
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, slStateIndex+1)
	err = c.settlementKV.Set(slStateIndexKey, b)
	if err != nil {
		return err
	}
	// Save latest height in memory and in store
	atomic.StoreUint64(&c.latestHeight, batch.EndHeight)
	return nil
}

func (c *HubClient) convertBatchtoSettlementBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) *settlement.Batch {
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

func (c *HubClient) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	b, err := c.settlementKV.Get(getKey(slStateIndex))
	c.logger.Debug("Retrieving batch from settlement layer", "SL state index", slStateIndex)
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

func getKey(key uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, key)
	return b
}
