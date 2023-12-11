package mock

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmp2p "github.com/tendermint/tendermint/p2p"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

// LayerClient is an extension of the base settlement layer client
// for usage in tests and local development.
type LayerClient struct {
	*settlement.BaseLayerClient
}

var _ settlement.LayerI = (*LayerClient)(nil)

// Init initializes the mock layer client.
func (m *LayerClient) Init(config settlement.Config, pubsub *pubsub.Server, logger log.Logger, kv store.KVStore, options ...settlement.Option) error {
	HubClientMock, err := newHubClient(config, pubsub, logger, kv)
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
	err = m.BaseLayerClient.Init(config, pubsub, logger, nil, options...)
	if err != nil {
		return err
	}
	return nil
}

// HubClient implements The HubClient interface
type HubClient struct {
	ProposerPubKey string
	slStateIndex   uint64
	logger         log.Logger
	pubsub         *pubsub.Server
	latestHeight   uint64
	settlementKV   store.KVStore
}

var _ settlement.HubClient = &HubClient{}

func newHubClient(config settlement.Config, pubsub *pubsub.Server, logger log.Logger, kv store.KVStore) (*HubClient, error) {
	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	slstore, proposer, err := initConfig(config)
	if err != nil {
		return nil, err
	}
	if kv != nil {
		slstore = kv
	}
	settlementKV := store.NewPrefixKV(slstore, settlementKVPrefix)
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
		ProposerPubKey: proposer,
		logger:         logger,
		pubsub:         pubsub,
		latestHeight:   latestHeight,
		slStateIndex:   slStateIndex,
		settlementKV:   settlementKV,
	}, nil
}

func initConfig(conf settlement.Config) (slstore store.KVStore, proposer string, err error) {
	if conf.KeyringHomeDir == "" {
		//init store
		slstore = store.NewDefaultInMemoryKVStore()
		//init proposer pub key
		if conf.ProposerPubKey != "" {
			proposer = conf.ProposerPubKey
		} else {
			_, proposerPubKey, err := crypto.GenerateEd25519Key(rand.Reader)
			if err != nil {
				return nil, "", err
			}
			pubKeybytes, err := proposerPubKey.Raw()
			if err != nil {
				return nil, "", err
			}

			proposer = hex.EncodeToString(pubKeybytes)
		}
	} else {
		slstore = store.NewDefaultKVStore(conf.KeyringHomeDir, "data", kvStoreDBName)
		fmt.Println("Setting proposarkeypath", "path", "config/priv_validator_key.json")
		proposerKeyPath := filepath.Join(conf.KeyringHomeDir, "config/priv_validator_key.json")
		key, err := tmp2p.LoadOrGenNodeKey(proposerKeyPath)
		if err != nil {
			return nil, "", err
		}
		proposer = hex.EncodeToString(key.PubKey().Bytes())
	}

	return
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
func (c *HubClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) {
	settlementBatch := c.convertBatchtoSettlementBatch(batch, daClient, daResult)
	c.saveBatch(settlementBatch)
	go func() {
		// sleep for 10 miliseconds to mimic a delay in batch acceptance
		time.Sleep(10 * time.Millisecond)
		err := c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewSettlementBatchAccepted{EndHeight: settlementBatch.EndHeight}, map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
		if err != nil {
			panic(err)
		}
	}()
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
	pubKeyBytes, err := hex.DecodeString(c.ProposerPubKey)
	if err != nil {
		return nil, err
	}
	var pubKey cryptotypes.PubKey = &ed25519.PubKey{Key: pubKeyBytes}
	return []*types.Sequencer{
		{
			PublicKey: pubKey,
			Status:    types.Proposer,
		},
	}, nil
}

func (c *HubClient) saveBatch(batch *settlement.Batch) {
	c.logger.Debug("Saving batch to settlement layer", "start height",
		batch.StartHeight, "end height", batch.EndHeight)
	b, err := json.Marshal(batch)
	if err != nil {
		panic(err)
	}
	// Save the batch to the next state index
	slStateIndex := atomic.LoadUint64(&c.slStateIndex)
	err = c.settlementKV.Set(getKey(slStateIndex+1), b)
	if err != nil {
		panic(err)
	}
	// Save SL state index in memory and in store
	atomic.StoreUint64(&c.slStateIndex, slStateIndex+1)
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, slStateIndex+1)
	err = c.settlementKV.Set(slStateIndexKey, b)
	if err != nil {
		panic(err)
	}
	// Save latest height in memory and in store
	atomic.StoreUint64(&c.latestHeight, batch.EndHeight)
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
