package settlement

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/utils"
	"github.com/tendermint/tendermint/libs/pubsub"
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

var _ LayerI = &BaseLayerClient{}

// WithHubClient is an option which sets the hub client.
func WithHubClient(hubClient HubClient) Option {
	return func(settlementClient LayerI) {
		settlementClient.(*BaseLayerClient).client = hubClient
	}
}

// Init is called once. it initializes the struct members.
func (b *BaseLayerClient) Init(config Config, pubsub *pubsub.Server, logger log.Logger, options ...Option) error {
	b.config = config
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

// SubmitBatch tries submitting the batch in an async broadcast mode to the settlement layer. Events are emitted on success or failure.
func (b *BaseLayerClient) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) {
	b.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	err := b.validateBatch(batch)
	if err != nil {
		panic(err)
	}
	b.client.PostBatch(batch, daClient, daResult)
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

func (b *BaseLayerClient) validateBatch(batch *types.Batch) error {
	if batch.StartHeight != atomic.LoadUint64(&b.latestHeight)+1 {
		return fmt.Errorf("batch start height != latest height + 1. StartHeight %d, lastetHeight %d", batch.StartHeight, atomic.LoadUint64(&b.latestHeight))
	}
	if batch.EndHeight < batch.StartHeight {
		return fmt.Errorf("batch end height must be greater than start height. EndHeight %d, StartHeight %d", batch.EndHeight, batch.StartHeight)
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
			eventData := event.Data().(*EventDataNewSettlementBatchAccepted)
			b.logger.Debug("received state update event", "latestHeight", eventData.EndHeight)
			atomic.StoreUint64(&b.latestHeight, eventData.EndHeight)
			// Emit new batch event
			newBatchEventData := &EventDataNewBatchAccepted{
				EndHeight:  eventData.EndHeight,
				StateIndex: eventData.StateIndex,
			}
			utils.SubmitEventOrPanic(b.ctx, b.pubsub, newBatchEventData,
				map[string][]string{EventTypeKey: {EventNewBatchAccepted}})
		case <-subscription.Cancelled():
			b.logger.Info("stateUpdatesHandler subscription canceled")
			return
		case <-b.ctx.Done():
			b.logger.Info("Context done. Exiting state update handler")
			return
		}
	}
}
