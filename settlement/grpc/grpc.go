package grpc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/libp2p/go-libp2p/core/crypto"
	tmp2p "github.com/tendermint/tendermint/p2p"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	rollapptypes "github.com/dymensionxyz/dymint/third_party/dymension/rollapp/types"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/pubsub"

	slmock "github.com/dymensionxyz/dymint/settlement/grpc/mockserv/proto"
)

// Client is an extension of the base settlement layer client
// for usage in tests and local development.
type Client struct {
	ctx            context.Context
	rollappID      string
	ProposerPubKey string
	slStateIndex   uint64
	logger         types.Logger
	pubsub         *pubsub.Server
	latestHeight   atomic.Uint64
	conn           *grpc.ClientConn
	sl             slmock.MockSLClient
	stopchan       chan struct{}
	refreshTime    int
}

var _ settlement.ClientI = (*Client)(nil)

// Init initializes the mock layer client.
func (c *Client) Init(config settlement.Config, pubsub *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
	ctx := context.Background()

	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	proposer, err := initConfig(config)
	if err != nil {
		return err
	}
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	logger.Debug("GRPC Dial ", "ip", config.SLGrpc.Host)

	conn, err := grpc.Dial(config.SLGrpc.Host+":"+strconv.Itoa(config.SLGrpc.Port), opts...)
	if err != nil {
		logger.Error("grpc sl connecting")
		return err
	}

	client := slmock.NewMockSLClient(conn)
	stopchan := make(chan struct{})

	index, err := client.GetIndex(ctx, &slmock.SLGetIndexRequest{})
	if err == nil {
		slStateIndex = index.GetIndex()
		var settlementBatch rollapptypes.MsgUpdateState
		batchReply, err := client.GetBatch(ctx, &slmock.SLGetBatchRequest{Index: slStateIndex})
		if err != nil {
			return err
		}
		err = json.Unmarshal(batchReply.GetBatch(), &settlementBatch)
		if err != nil {
			return errors.New("error unmarshalling batch")
		}
		latestHeight = settlementBatch.StartHeight + settlementBatch.NumBlocks - 1
	}
	logger.Debug("Starting grpc SL ", "index", slStateIndex)

	c.rollappID = config.RollappID
	c.ProposerPubKey = proposer
	c.logger = logger
	c.ctx = ctx
	c.pubsub = pubsub
	c.slStateIndex = slStateIndex
	c.conn = conn
	c.sl = client
	c.stopchan = stopchan
	c.refreshTime = config.SLGrpc.RefreshTime
	c.latestHeight.Store(latestHeight)

	return nil
}

func initConfig(conf settlement.Config) (proposer string, err error) {
	if conf.KeyringHomeDir == "" {
		if conf.ProposerPubKey != "" {
			proposer = conf.ProposerPubKey
		} else {
			_, proposerPubKey, err := crypto.GenerateEd25519Key(rand.Reader)
			if err != nil {
				return "", err
			}
			pubKeybytes, err := proposerPubKey.Raw()
			if err != nil {
				return "", err
			}

			proposer = hex.EncodeToString(pubKeybytes)
		}
	} else {
		proposerKeyPath := filepath.Join(conf.KeyringHomeDir, "config/priv_validator_key.json")
		key, err := tmp2p.LoadOrGenNodeKey(proposerKeyPath)
		if err != nil {
			return "", err
		}
		proposer = hex.EncodeToString(key.PubKey().Bytes())
	}

	return
}

// Start starts the mock client
func (c *Client) Start() error {
	c.logger.Info("Starting grpc mock settlement")

	go func() {
		tick := time.NewTicker(time.Duration(c.refreshTime) * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-c.stopchan:
				// stop
				return
			case <-tick.C:
				index, err := c.sl.GetIndex(c.ctx, &slmock.SLGetIndexRequest{})
				if err == nil {
					if c.slStateIndex < index.GetIndex() {
						c.logger.Info("Simulating new batch event")

						time.Sleep(10 * time.Millisecond)
						b, err := c.retrieveBatchAtStateIndex(index.GetIndex())
						if err != nil {
							panic(err)
						}
						err = c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewBatchAccepted{EndHeight: b.EndHeight}, settlement.EventNewBatchAcceptedList)
						if err != nil {
							panic(err)
						}
						c.slStateIndex = index.GetIndex()
					}
				}
			}
		}
	}()
	return nil
}

// Stop stops the mock client
func (c *Client) Stop() error {
	c.logger.Info("Stopping grpc mock settlement")
	close(c.stopchan)
	return nil
}

// SubmitBatch saves the batch to the kv store
func (c *Client) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	settlementBatch := c.convertBatchtoSettlementBatch(batch, daResult)
	err := c.saveBatch(settlementBatch)
	if err != nil {
		return err
	}

	time.Sleep(10 * time.Millisecond) // mimic a delay in batch acceptance
	err = c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewBatchAccepted{EndHeight: settlementBatch.EndHeight}, settlement.EventNewBatchAcceptedList)
	if err != nil {
		return err
	}
	return nil
}

// GetLatestBatch returns the latest batch from the kv store
func (c *Client) GetLatestBatch() (*settlement.ResultRetrieveBatch, error) {
	c.logger.Info("GetLatestBatch grpc", "index", c.slStateIndex)
	batchResult, err := c.GetBatchAtIndex(atomic.LoadUint64(&c.slStateIndex))
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

func (c *Client) GetHeightState(index uint64) (*settlement.ResultGetHeightState, error) {
	panic("hub grpc client get height state is not implemented: implement me") // TODO: impl
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
	c.logger.Debug("Saving batch to grpc settlement layer", "start height",
		batch.StartHeight, "end height", batch.EndHeight)
	b, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	// Save the batch to the next state index
	c.logger.Debug("Saving batch to grpc settlement layer", "index", c.slStateIndex+1)
	setBatchReply, err := c.sl.SetBatch(c.ctx, &slmock.SLSetBatchRequest{Index: c.slStateIndex + 1, Batch: b})
	if err != nil {
		return err
	}
	if setBatchReply.GetResult() != c.slStateIndex+1 {
		return err
	}

	c.slStateIndex = setBatchReply.GetResult()

	setIndexReply, err := c.sl.SetIndex(c.ctx, &slmock.SLSetIndexRequest{Index: c.slStateIndex})
	if err != nil || setIndexReply.GetIndex() != c.slStateIndex {
		return err
	}
	c.logger.Debug("Setting grpc SL Index to ", "index", setIndexReply.GetIndex())
	// Save latest height in memory and in store
	c.latestHeight.Store(batch.EndHeight)
	return nil
}

func (c *Client) convertBatchtoSettlementBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) *settlement.Batch {
	settlementBatch := &settlement.Batch{
		StartHeight: batch.StartHeight(),
		EndHeight:   batch.EndHeight(),
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

func (c *Client) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	c.logger.Debug("Retrieving batch from grpc settlement layer", "SL state index", slStateIndex)

	getBatchReply, err := c.sl.GetBatch(c.ctx, &slmock.SLGetBatchRequest{Index: slStateIndex})
	if err != nil {
		return nil, gerrc.ErrNotFound
	}
	b := getBatchReply.GetBatch()
	if b == nil {
		return nil, gerrc.ErrNotFound
	}
	var settlementBatch settlement.Batch
	err = json.Unmarshal(b, &settlementBatch)
	if err != nil {
		return nil, errors.New("error unmarshalling batch")
	}
	batchResult := settlement.ResultRetrieveBatch{
		ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess, StateIndex: slStateIndex},
		Batch:      &settlementBatch,
	}
	return &batchResult, nil
}
