package local

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/dymensionxyz/dymint/gerr"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmp2p "github.com/tendermint/tendermint/p2p"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	rollapptypes "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/pubsub"
)

const kvStoreDBName = "settlement"

var (
	settlementKVPrefix = []byte{0}
	slStateIndexKey    = []byte("slStateIndex") // used to recover after reboot
)

// LayerClient is an extension of the base settlement layer client
// for usage in tests and local development.
type LayerClient struct {
	*settlement.BaseLayerClient
}

var _ settlement.LayerI = (*LayerClient)(nil)

// Init initializes the mock layer client.
func (m *LayerClient) Init(config settlement.Config, pubsub *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
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

// HubClient implements The HubClient interface
type HubClient struct {
	ProposerPubKey string
	logger         types.Logger
	pubsub         *pubsub.Server

	mu           sync.Mutex // keep the following in sync with *each other*
	slStateIndex uint64
	latestHeight uint64
	settlementKV store.KVStore
}

var _ settlement.HubClient = &HubClient{}

func newHubClient(config settlement.Config, pubsub *pubsub.Server, logger types.Logger) (*HubClient, error) {
	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	slstore, proposer, err := initConfig(config)
	if err != nil {
		return nil, err
	}
	settlementKV := store.NewPrefixKV(slstore, settlementKVPrefix)
	b, err := settlementKV.Get(slStateIndexKey)
	if err == nil {
		slStateIndex = binary.BigEndian.Uint64(b)
		// Get the latest height from the stateIndex
		var settlementBatch rollapptypes.MsgUpdateState
		b, err := settlementKV.Get(keyFromIndex(slStateIndex))
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
		// init store
		slstore = store.NewDefaultInMemoryKVStore()
		// init proposer pub key
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
func (c *HubClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	settlementBatch := convertBatchToSettlementBatch(batch, daResult)
	c.saveBatch(settlementBatch)
	go func() {
		time.Sleep(10 * time.Millisecond) // mimic a delay in batch acceptance
		err := c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewBatchAccepted{EndHeight: settlementBatch.EndHeight}, settlement.EventNewBatchAcceptedList)
		if err != nil {
			panic(err)
		}
	}()
	return nil
}

// GetLatestBatch returns the latest batch from the kv store
func (c *HubClient) GetLatestBatch(rollappID string) (*settlement.ResultRetrieveBatch, error) {
	c.mu.Lock()
	ix := c.slStateIndex
	c.mu.Unlock()
	batchResult, err := c.GetBatchAtIndex(rollappID, ix)
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
			ResultBase: settlement.ResultBase{Code: settlement.StatusError, Message: err.Error()},
		}, err
	}
	return batchResult, nil
}

func (c *HubClient) GetHeightState(h uint64) (*settlement.ResultGetHeightState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO: optimize (binary search, or just make another index)
	for i := c.slStateIndex; i > 0; i-- {
		b, err := c.GetBatchAtIndex("", i)
		if err != nil {
			panic(err)
		}
		if b.StartHeight <= h && b.EndHeight >= h {
			return &settlement.ResultGetHeightState{
				ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess},
				State: settlement.State{
					StateIndex: i,
				},
			}, nil
		}
	}
	return nil, gerr.ErrNotFound // TODO: need to return a cosmos specific error?
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
	c.logger.Debug("Saving batch to settlement layer.", "start height",
		batch.StartHeight, "end height", batch.EndHeight)

	b, err := json.Marshal(batch)
	if err != nil {
		panic(err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Save the batch to the next state index
	c.slStateIndex++
	err = c.settlementKV.Set(keyFromIndex(c.slStateIndex), b)
	if err != nil {
		panic(err)
	}
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, c.slStateIndex)
	err = c.settlementKV.Set(slStateIndexKey, b)
	if err != nil {
		panic(err)
	}
	c.latestHeight = batch.EndHeight
}

func (c *HubClient) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	b, err := c.settlementKV.Get(keyFromIndex(slStateIndex))
	c.logger.Debug("Retrieving batch from settlement layer.", "SL state index", slStateIndex)
	if err != nil {
		return nil, gerr.ErrNotFound
	}
	var settlementBatch settlement.Batch
	err = json.Unmarshal(b, &settlementBatch)
	if err != nil {
		return nil, errors.New("unmarshalling batch")
	}
	batchResult := settlement.ResultRetrieveBatch{
		ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess, StateIndex: slStateIndex},
		Batch:      &settlementBatch,
	}
	return &batchResult, nil
}

func convertBatchToSettlementBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) *settlement.Batch {
	settlementBatch := &settlement.Batch{
		StartHeight: batch.StartHeight,
		EndHeight:   batch.EndHeight,
		MetaData: &settlement.BatchMetaData{
			DA: &da.DASubmitMetaData{
				Height: daResult.SubmitMetaData.Height,
				Client: daResult.SubmitMetaData.Client,
			},
		},
	}
	for _, block := range batch.Blocks {
		settlementBatch.AppHashes = append(settlementBatch.AppHashes, block.Header.AppHash)
	}
	return settlementBatch
}

func keyFromIndex(ix uint64) []byte {
	b := make([]byte, 8)
	b = append(b, byte('i'))
	binary.BigEndian.PutUint64(b, ix)
	return b
}
