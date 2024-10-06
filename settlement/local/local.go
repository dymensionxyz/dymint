package local

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/pubsub"
	tmp2p "github.com/tendermint/tendermint/p2p"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

const kvStoreDBName = "settlement"

var (
	settlementKVPrefix = []byte{0}
	slStateIndexKey    = []byte("slStateIndex") // used to recover after reboot
)

// Client is an extension of the base settlement layer client
// for usage in tests and local development.
type Client struct {
	rollappID      string
	ProposerPubKey string
	logger         types.Logger
	pubsub         *pubsub.Server

	mu           sync.Mutex // keep the following in sync with *each other*
	slStateIndex uint64
	latestHeight uint64
	settlementKV store.KV
}

var _ settlement.ClientI = (*Client)(nil)

// Init initializes the mock layer client.
func (c *Client) Init(config settlement.Config, rollappId string, pubsub *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
	slstore, proposer, err := initConfig(config)
	if err != nil {
		return err
	}

	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	settlementKV := store.NewPrefixKV(slstore, settlementKVPrefix)
	b, err := settlementKV.Get(slStateIndexKey)
	if err == nil {
		slStateIndex = binary.BigEndian.Uint64(b)
		// Get the latest height from the stateIndex
		var settlementBatch rollapptypes.MsgUpdateState
		b, err := settlementKV.Get(keyFromIndex(slStateIndex))
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &settlementBatch)
		if err != nil {
			return errors.New("error unmarshalling batch")
		}
		latestHeight = settlementBatch.StartHeight + settlementBatch.NumBlocks - 1
	}
	c.rollappID = rollappId
	c.ProposerPubKey = proposer
	c.logger = logger
	c.pubsub = pubsub
	c.latestHeight = latestHeight
	c.slStateIndex = slStateIndex
	c.settlementKV = settlementKV
	return nil
}

func initConfig(conf settlement.Config) (slstore store.KV, proposer string, err error) {
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
		if conf.ProposerPubKey != "" {
			proposer = conf.ProposerPubKey
		} else {
			proposerKeyPath := filepath.Join(conf.KeyringHomeDir, "config/priv_validator_key.json")
			key, err := tmp2p.LoadOrGenNodeKey(proposerKeyPath)
			if err != nil {
				return nil, "", fmt.Errorf("loading sequencer pubkey: %w", err)
			}
			proposer = hex.EncodeToString(key.PubKey().Bytes())
		}
	}

	return
}

// Start starts the mock client
func (c *Client) Start() error {
	return nil
}

// Stop stops the mock client
func (c *Client) Stop() error {
	return c.settlementKV.Close()
}

// PostBatch saves the batch to the kv store
func (c *Client) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	settlementBatch := c.convertBatchToSettlementBatch(batch, daResult)
	err := c.saveBatch(settlementBatch)
	if err != nil {
		return err
	}

	time.Sleep(100 * time.Millisecond) // mimic a delay in batch acceptance
	ctx := context.Background()
	uevent.MustPublish(ctx, c.pubsub, settlement.EventDataNewBatchAccepted{EndHeight: settlementBatch.EndHeight}, settlement.EventNewBatchAcceptedList)

	return nil
}

// GetLatestBatch returns the latest batch from the kv store
func (c *Client) GetLatestBatch() (*settlement.ResultRetrieveBatch, error) {
	c.mu.Lock()
	ix := c.slStateIndex
	c.mu.Unlock()
	batchResult, err := c.GetBatchAtIndex(ix)
	if err != nil {
		return nil, err
	}
	return batchResult, nil
}

// GetBatchAtIndex returns the batch at the given index
func (c *Client) GetBatchAtIndex(index uint64) (*settlement.ResultRetrieveBatch, error) {
	batchResult, err := c.retrieveBatchAtStateIndex(index)
	if err != nil {
		return &settlement.ResultRetrieveBatch{
			ResultBase: settlement.ResultBase{Code: settlement.StatusError, Message: err.Error()},
		}, err
	}
	return batchResult, nil
}

func (c *Client) GetHeightState(h uint64) (*settlement.ResultGetHeightState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// TODO: optimize (binary search, or just make another index)
	for i := c.slStateIndex; i > 0; i-- {
		b, err := c.GetBatchAtIndex(i)
		if err != nil {
			return nil, err
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
	return nil, gerrc.ErrNotFound // TODO: need to return a cosmos specific error?
}

// GetProposer implements settlement.ClientI.
func (c *Client) GetProposer() *types.Sequencer {
	pubKeyBytes, err := hex.DecodeString(c.ProposerPubKey)
	if err != nil {
		return nil
	}
	var pubKey cryptotypes.PubKey = &ed25519.PubKey{Key: pubKeyBytes}
	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
	if err != nil {
		c.logger.Error("Error converting to tendermint pubkey", "err", err)
		return nil
	}
	return types.NewSequencer(tmPubKey, pubKey.Address().String())
}

// GetAllSequencers implements settlement.ClientI.
func (c *Client) GetAllSequencers() ([]types.Sequencer, error) {
	return c.GetBondedSequencers()
}

// GetBondedSequencers implements settlement.ClientI.
func (c *Client) GetBondedSequencers() ([]types.Sequencer, error) {
	return []types.Sequencer{*c.GetProposer()}, nil
}

// CheckRotationInProgress implements settlement.ClientI.
func (c *Client) CheckRotationInProgress() (*types.Sequencer, error) {
	return nil, nil
}

func (c *Client) saveBatch(batch *settlement.Batch) error {
	c.logger.Debug("Saving batch to settlement layer.", "start height",
		batch.StartHeight, "end height", batch.EndHeight)

	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Save the batch to the next state index
	c.slStateIndex++
	err = c.settlementKV.Set(keyFromIndex(c.slStateIndex), b)
	if err != nil {
		return err
	}
	b = make([]byte, 8)
	binary.BigEndian.PutUint64(b, c.slStateIndex)
	err = c.settlementKV.Set(slStateIndexKey, b)
	if err != nil {
		return err
	}
	c.latestHeight = batch.EndHeight
	return nil
}

func (c *Client) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	b, err := c.settlementKV.Get(keyFromIndex(slStateIndex))
	c.logger.Debug("Retrieving batch from settlement layer.", "SL state index", slStateIndex)
	if err != nil {
		return nil, gerrc.ErrNotFound
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

func (c *Client) convertBatchToSettlementBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) *settlement.Batch {
	bds := []settlement.BlockDescriptor{}
	for _, block := range batch.Blocks {
		bd := settlement.BlockDescriptor{
			Height:    block.Header.Height,
			StateRoot: block.Header.AppHash[:],
			Timestamp: block.Header.GetTimestamp(),
		}
		bds = append(bds, bd)
	}

	settlementBatch := &settlement.Batch{
		Sequencer:   c.GetProposer().SettlementAddress,
		StartHeight: batch.StartHeight(),
		EndHeight:   batch.EndHeight(),
		MetaData: &settlement.BatchMetaData{
			DA: &da.DASubmitMetaData{
				Height: daResult.SubmitMetaData.Height,
				Client: daResult.SubmitMetaData.Client,
			},
		},
		BlockDescriptors: bds,
	}

	return settlementBatch
}

func keyFromIndex(ix uint64) []byte {
	b := make([]byte, 8)
	b = append(b, byte('i'))
	binary.BigEndian.PutUint64(b, ix)
	return b
}
