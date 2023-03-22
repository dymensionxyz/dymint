package settlement

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

const (
	defaultBatchSize = 5
)

// BaseLayerClient is intended only for usage in tests.
type BaseLayerClient struct {
	logger         log.Logger
	pubsub         *pubsub.Server
	latestHeight   uint64
	sequencersList []*types.Sequencer
	config         Config
	ctx            context.Context
	cancel         context.CancelFunc
	client         HubClient
}

// Config for the BaseLayerClient
type Config struct {
	BatchSize uint64 `json:"batch_size"`
	RollappID string `json:"rollapp_id"`
}

var _ LayerClient = &BaseLayerClient{}

// WithHubClient is an option which sets the hub client.
func WithHubClient(hubClient HubClient) Option {
	return func(settlementClient LayerClient) {
		settlementClient.(*BaseLayerClient).client = hubClient
	}
}

// Init is called once. it initializes the struct members.
func (b *BaseLayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger, options ...Option) error {
	c, err := b.getConfig(config)
	if err != nil {
		return err
	}
	b.config = *c
	b.pubsub = pubsub
	b.logger = logger
	b.ctx, b.cancel = context.WithCancel(context.Background())
	// Apply options
	for _, apply := range options {
		apply(b)
	}

	latestBatch, err := b.RetrieveBatch()
	var endHeight uint64
	if err != nil {
		if err == ErrBatchNotFound {
			endHeight = 0
		} else {
			return err
		}
	} else {
		endHeight = latestBatch.EndHeight
	}
	b.latestHeight = endHeight
	b.logger.Info("Updated latest height from settlement layer", "latestHeight", endHeight)
	b.sequencersList, err = b.fetchSequencersList()
	if err != nil {
		if err == ErrNoSequencerForRollapp {
			panic(err)
		}
		return err
	}
	b.logger.Info("Updated sequencers list from settlement layer", "sequencersList", b.sequencersList)

	return nil
}

// Start is called once, after init. It initializes the query client.
func (b *BaseLayerClient) Start() error {
	b.logger.Debug("settlement Layer Client starting.")

	// Wait until the state updates handler is ready
	ready := make(chan bool, 1)
	go b.stateUpdatesHandler(ready)
	<-ready

	err := b.client.Start()
	if err != nil {
		return err
	}

	return nil
}

// Stop is called once, after Start.
func (b *BaseLayerClient) Stop() error {
	b.logger.Info("Settlement Layer Client stopping")
	err := b.client.Stop()
	if err != nil {
		return err
	}
	b.cancel()
	return nil
}

// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (b *BaseLayerClient) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) *ResultSubmitBatch {
	b.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	err := b.validateBatch(batch)
	if err != nil {
		return &ResultSubmitBatch{
			BaseResult: BaseResult{Code: StatusError, Message: err.Error()},
		}
	}
	txResp, err := b.client.PostBatch(batch, daClient, daResult)
	if err != nil || txResp.GetCode() != 0 {
		b.logger.Error("Error sending batch to settlement layer", "error", err)
		return &ResultSubmitBatch{
			BaseResult: BaseResult{Code: StatusError, Message: err.Error()},
		}
	}
	b.logger.Info("Successfully submitted batch to settlement layer", "tx hash", txResp.GetTxHash())
	return &ResultSubmitBatch{
		BaseResult: BaseResult{Code: StatusSuccess},
	}
}

// RetrieveBatch Gets the batch which contains the given slHeight. Empty slHeight returns the latest batch.
func (b *BaseLayerClient) RetrieveBatch(stateIndex ...uint64) (*ResultRetrieveBatch, error) {
	var resultRetrieveBatch *ResultRetrieveBatch
	var err error
	if len(stateIndex) == 0 {
		b.logger.Debug("Getting latest batch from settlement layer")
		resultRetrieveBatch, err = b.client.GetLatestBatch(b.config.RollappID)
		if err != nil {
			return nil, err
		}
	} else if len(stateIndex) == 1 {
		b.logger.Debug("Getting batch from settlement layer", "state index", stateIndex)
		resultRetrieveBatch, err = b.client.GetBatchAtIndex(b.config.RollappID, stateIndex[0])
		if err != nil {
			return nil, err
		}
	}
	return resultRetrieveBatch, nil
}

// GetSequencersList returns the current list of sequencers from the settlement layer
func (b *BaseLayerClient) GetSequencersList() []*types.Sequencer {
	return b.sequencersList
}

// GetProposer returns the sequencer which is currently the proposer
func (b *BaseLayerClient) GetProposer() *types.Sequencer {
	for _, sequencer := range b.sequencersList {
		if sequencer.Status == types.Proposer {
			return sequencer
		}
	}
	return nil
}

func (b *BaseLayerClient) fetchSequencersList() ([]*types.Sequencer, error) {
	sequencers, err := b.client.GetSequencers(b.config.RollappID)
	if err != nil {
		return nil, err
	}
	return sequencers, nil
}

func (b *BaseLayerClient) decodeConfig(config []byte) (*Config, error) {
	var c Config
	err := json.Unmarshal(config, &c)
	return &c, err
}

func (b *BaseLayerClient) getConfig(config []byte) (*Config, error) {
	var c *Config
	if len(config) > 0 {
		var err error
		c, err = b.decodeConfig(config)
		if err != nil {
			return nil, err
		}
		if c.BatchSize == 0 {
			c.BatchSize = defaultBatchSize
		}
	} else {
		c = &Config{
			BatchSize: defaultBatchSize,
		}
	}
	return c, nil
}

func (b *BaseLayerClient) validateBatch(batch *types.Batch) error {
	if batch.StartHeight != atomic.LoadUint64(&b.latestHeight)+1 {
		return errors.New("batch start height must be last height + 1")
	}
	if batch.EndHeight < batch.StartHeight {
		return errors.New("batch end height must be greater or equal to start height")
	}
	return nil
}

func (b *BaseLayerClient) stateUpdatesHandler(ready chan bool) {
	b.logger.Info("started state updates handler loop")
	subscription, err := b.pubsub.Subscribe(b.ctx, "stateUpdatesHandler", EventQueryNewSettlementBatchAccepted)
	if err != nil {
		b.logger.Error("failed to subscribe to state update events", "error", err)
		panic(err)
	}
	ready <- true
	for {
		select {
		case event := <-subscription.Out():
			b.logger.Debug("received state update event", "eventData", event.Data())
			eventData := event.Data().(*EventDataNewSettlementBatchAccepted)
			atomic.StoreUint64(&b.latestHeight, eventData.EndHeight)
		case <-subscription.Cancelled():
			b.logger.Info("subscription canceled")
			return
		case <-b.ctx.Done():
			return
		}
	}
}
