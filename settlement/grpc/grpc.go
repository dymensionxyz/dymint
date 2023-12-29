package grpc

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	tmp2p "github.com/tendermint/tendermint/p2p"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"

	"github.com/tendermint/tendermint/libs/pubsub"

	slmock "github.com/dymensionxyz/dymint/settlement/grpc/mockserv/proto"
)

// LayerClient is an extension of the base settlement layer client
// for usage in tests and local development.
type LayerClient struct {
	*settlement.BaseLayerClient
}

var _ settlement.LayerI = (*LayerClient)(nil)

// Init initializes the mock layer client.
func (m *LayerClient) Init(config settlement.Config, pubsub *pubsub.Server, logger log.Logger, options ...settlement.Option) error {
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
type HubGrpcClient struct {
	ctx            context.Context
	ProposerPubKey string
	slStateIndex   uint64
	logger         log.Logger
	pubsub         *pubsub.Server
	latestHeight   uint64
	conn           *grpc.ClientConn
	sl             slmock.MockSLClient
	stopchan       chan struct{}
}

var _ settlement.HubClient = &HubGrpcClient{}

func newHubClient(config settlement.Config, pubsub *pubsub.Server, logger log.Logger) (*HubGrpcClient, error) {
	ctx, _ := context.WithCancel(context.Background())

	latestHeight := uint64(0)
	slStateIndex := uint64(0)
	proposer, err := initConfig(config)
	if err != nil {
		return nil, err
	}
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	logger.Debug("GRPC Dial ", "ip", config.SLGrpc.Host)

	conn, err := grpc.Dial(config.SLGrpc.Host+":"+strconv.Itoa(config.SLGrpc.Port), opts...)
	if err != nil {
		logger.Error("Error grpc sl connecting")
		return nil, err
	}

	client := slmock.NewMockSLClient(conn)
	stopchan := make(chan struct{})

	index, err := client.GetIndex(ctx, &slmock.SLGetIndexRequest{})
	if err == nil {
		slStateIndex = index.GetIndex()
		var settlementBatch rollapptypes.MsgUpdateState
		batchReply, err := client.GetBatch(ctx, &slmock.SLGetBatchRequest{Index: slStateIndex})
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(batchReply.GetBatch(), &settlementBatch)
		if err != nil {
			return nil, errors.New("error unmarshalling batch")
		}
		latestHeight = settlementBatch.StartHeight + settlementBatch.NumBlocks - 1
	}
	logger.Debug("Starting grpc SL ", "index", slStateIndex)

	return &HubGrpcClient{
		ctx:            ctx,
		ProposerPubKey: proposer,
		logger:         logger,
		pubsub:         pubsub,
		latestHeight:   latestHeight,
		slStateIndex:   slStateIndex,
		conn:           conn,
		sl:             client,
		stopchan:       stopchan,
	}, nil
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
func (c *HubGrpcClient) Start() error {
	c.logger.Info("Starting grpc mock settlement")
	tick := time.Tick(1 * time.Second)
	// Keep trying until we're timed out or got a result or got an error
	go func() {
		for {
			select {
			case <-c.stopchan:
				// stop
				return
			case <-tick:
				index, err := c.sl.GetIndex(c.ctx, &slmock.SLGetIndexRequest{})
				if err == nil {
					if c.slStateIndex < index.GetIndex() {
						c.logger.Info("Simulating new batch event")

						time.Sleep(10 * time.Millisecond)
						b, err := c.retrieveBatchAtStateIndex(index.GetIndex())
						if err != nil {
							panic(err)
						}
						err = c.pubsub.PublishWithEvents(context.Background(), &settlement.EventDataNewSettlementBatchAccepted{EndHeight: b.EndHeight}, map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
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
func (c *HubGrpcClient) Stop() error {
	c.logger.Info("Stopping grpc mock settlement")
	close(c.stopchan)
	return nil
}

// PostBatch saves the batch to the kv store
func (c *HubGrpcClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) {
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
func (c *HubGrpcClient) GetLatestBatch(rollappID string) (*settlement.ResultRetrieveBatch, error) {
	c.logger.Info("GetLatestBatch grpc", "index", c.slStateIndex)
	batchResult, err := c.GetBatchAtIndex(rollappID, atomic.LoadUint64(&c.slStateIndex))
	if err != nil {
		return nil, err
	}
	return batchResult, nil
}

// GetBatchAtIndex returns the batch at the given index
func (c *HubGrpcClient) GetBatchAtIndex(rollappID string, index uint64) (*settlement.ResultRetrieveBatch, error) {
	batchResult, err := c.retrieveBatchAtStateIndex(index)
	if err != nil {
		return &settlement.ResultRetrieveBatch{
			BaseResult: settlement.BaseResult{Code: settlement.StatusError, Message: err.Error()},
		}, err
	}
	return batchResult, nil
}

// GetSequencers returns a list of sequencers. Currently only returns a single sequencer
func (c *HubGrpcClient) GetSequencers(rollappID string) ([]*types.Sequencer, error) {
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

func (c *HubGrpcClient) saveBatch(batch *settlement.Batch) {
	c.logger.Debug("Saving batch to grpc settlement layer", "start height",
		batch.StartHeight, "end height", batch.EndHeight)
	b, err := json.Marshal(batch)
	if err != nil {
		panic(err)
	}
	// Save the batch to the next state index
	c.logger.Debug("Saving batch to grpc settlement layer", "index", c.slStateIndex+1)
	setBatchReply, err := c.sl.SetBatch(c.ctx, &slmock.SLSetBatchRequest{Index: c.slStateIndex + 1, Batch: b})
	if err != nil {
		panic(err)
	}
	if setBatchReply.GetResult() != c.slStateIndex+1 {
		panic(err)
	}

	c.slStateIndex = setBatchReply.GetResult()

	setIndexReply, err := c.sl.SetIndex(c.ctx, &slmock.SLSetIndexRequest{Index: c.slStateIndex})
	if err != nil || setIndexReply.GetIndex() != c.slStateIndex {
		panic(err)
	}
	c.logger.Debug("Setting grpc SL Index to ", "index", setIndexReply.GetIndex())
	// Save latest height in memory and in store
	atomic.StoreUint64(&c.latestHeight, batch.EndHeight)
}

func (c *HubGrpcClient) convertBatchtoSettlementBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) *settlement.Batch {
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

func (c *HubGrpcClient) retrieveBatchAtStateIndex(slStateIndex uint64) (*settlement.ResultRetrieveBatch, error) {
	c.logger.Debug("Retrieving batch from grpc settlement layer", "SL state index", slStateIndex)

	getBatchReply, err := c.sl.GetBatch(c.ctx, &slmock.SLGetBatchRequest{Index: slStateIndex})
	if err != nil {
		return nil, settlement.ErrBatchNotFound
	}
	b := getBatchReply.GetBatch()
	if b == nil {
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
