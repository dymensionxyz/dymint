package dymension

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"
	"github.com/google/uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/pkg/errors"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sequencertypes "github.com/dymensionxyz/dymension/x/sequencer/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/utils"
	"github.com/hashicorp/go-multierror"
	"github.com/tendermint/tendermint/libs/pubsub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	addressPrefix     = "dym"
	dymRollappVersion = 0
	defaultGasLimit   = 300000
)

const (
	eventStateUpdate          = "state_update.rollapp_id='%s'"
	eventSequencersListUpdate = "sequencers_list_update.rollapp_id='%s'"
)

const (
	batchRetryDelay        = 10 * time.Second
	batchRetryMaxDelay     = 1 * time.Minute
	batchAcceptanceTimeout = 120 * time.Second
	batchRetryAttempts     = 10
)

const (
	postBatchSubscriberPrefix = "postBatchSubscriber"
)

// LayerClient is intended only for usage in tests.
type LayerClient struct {
	*settlement.BaseLayerClient
}

var _ settlement.LayerI = &LayerClient{}

// Init is called once. it initializes the struct members.
func (dlc *LayerClient) Init(config settlement.Config, pubsub *pubsub.Server, logger log.Logger, options ...settlement.Option) error {
	DymensionCosmosClient, err := newDymensionHubClient(config, pubsub, logger)
	if err != nil {
		return err
	}
	baseOptions := []settlement.Option{
		settlement.WithHubClient(DymensionCosmosClient),
	}
	if options == nil {
		options = baseOptions
	} else {
		options = append(baseOptions, options...)
	}
	dlc.BaseLayerClient = &settlement.BaseLayerClient{}
	err = dlc.BaseLayerClient.Init(config, pubsub, logger, options...)
	if err != nil {
		return err
	}
	return nil
}

// HubClient is the client for the Dymension Hub.
type HubClient struct {
	config               *settlement.Config
	logger               log.Logger
	pubsub               *pubsub.Server
	client               CosmosClient
	ctx                  context.Context
	cancel               context.CancelFunc
	rollappQueryClient   rollapptypes.QueryClient
	sequencerQueryClient sequencertypes.QueryClient
	protoCodec           *codec.ProtoCodec
	eventMap             map[string]string
	// channel for getting notified when a batch is accepted by the settlement layer.
	// only one batch of a specific height can get accepted and we can are currently sending only one batch at a time.
	// for that reason it's safe to assume that if a batch is accepted, it refers to the last batch we've sent.
	batchRetryAttempts     uint
	batchRetryDelay        time.Duration
	batchAcceptanceTimeout time.Duration
}

var _ settlement.HubClient = &HubClient{}

// Option is a function that configures the HubClient.
type Option func(*HubClient)

// WithCosmosClient is an option that sets the CosmosClient.
func WithCosmosClient(cosmosClient CosmosClient) Option {
	return func(d *HubClient) {
		d.client = cosmosClient
	}
}

// WithBatchRetryAttempts is an option that sets the number of attempts to retry sending a batch to the settlement layer.
func WithBatchRetryAttempts(batchRetryAttempts uint) Option {
	return func(d *HubClient) {
		d.batchRetryAttempts = batchRetryAttempts
	}
}

// WithBatchAcceptanceTimeout is an option that sets the timeout for waiting for a batch to be accepted by the settlement layer.
func WithBatchAcceptanceTimeout(batchAcceptanceTimeout time.Duration) Option {
	return func(d *HubClient) {
		d.batchAcceptanceTimeout = batchAcceptanceTimeout
	}
}

// WithBatchRetryDelay is an option that sets the delay between batch retry attempts.
func WithBatchRetryDelay(batchRetryDelay time.Duration) Option {
	return func(d *HubClient) {
		d.batchRetryDelay = batchRetryDelay
	}
}

func newDymensionHubClient(config settlement.Config, pubsub *pubsub.Server, logger log.Logger, options ...Option) (*HubClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, config.RollappID):          settlement.EventNewSettlementBatchAccepted,
		fmt.Sprintf(eventSequencersListUpdate, config.RollappID): settlement.EventSequencersListUpdated,
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	dymesionHubClient := &HubClient{
		config:                 &config,
		logger:                 logger,
		pubsub:                 pubsub,
		ctx:                    ctx,
		cancel:                 cancel,
		eventMap:               eventMap,
		protoCodec:             protoCodec,
		batchRetryAttempts:     batchRetryAttempts,
		batchAcceptanceTimeout: batchAcceptanceTimeout,
		batchRetryDelay:        batchRetryDelay,
	}

	for _, option := range options {
		option(dymesionHubClient)
	}

	if dymesionHubClient.client == nil {
		client, err := cosmosclient.New(
			ctx,
			getCosmosClientOptions(&config)...,
		)
		if err != nil {
			return nil, err
		}
		dymesionHubClient.client = NewCosmosClient(client)
	}
	dymesionHubClient.rollappQueryClient = dymesionHubClient.client.GetRollappClient()
	dymesionHubClient.sequencerQueryClient = dymesionHubClient.client.GetSequencerClient()

	return dymesionHubClient, nil
}

// Start starts the HubClient.
func (d *HubClient) Start() error {
	err := d.client.StartEventListener()
	if err != nil {
		return err
	}
	go d.eventHandler()
	return nil

}

// Stop stops the HubClient.
func (d *HubClient) Stop() error {
	d.cancel()
	err := d.client.StopEventListener()
	if err != nil {
		return err
	}

	return nil
}

// PostBatch posts a batch to the Dymension Hub. it tries to post the batch until it is accepted by the settlement layer.
// it emits success and failure events to the event bus accordingly.
func (d *HubClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) {
	msgUpdateState, err := d.convertBatchToMsgUpdateState(batch, daClient, daResult)
	if err != nil {
		panic(err)
	}

	postBatchSubscriberClient := fmt.Sprintf("%s-%d-%s", postBatchSubscriberPrefix, batch.StartHeight, uuid.New().String())
	subscription, err := d.pubsub.Subscribe(d.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		d.logger.Error("failed to subscribe to state update events", "err", err)
		panic(err)
	}

	//nolint:errcheck
	defer d.pubsub.Unsubscribe(d.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted)

	// Try submitting the batch to the settlement layer. If submission (i.e only submission, not acceptance) fails we emit an unhealthy event
	// and try again in the next loop. If submission succeeds we wait for the batch to be accepted by the settlement layer.
	// If it is not accepted we emit an unhealthy event and start again the submission loop.
	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			// Try submitting the batch
			err := d.submitBatch(msgUpdateState)
			if err != nil {
				d.logger.Error("Failed submitting batch to settlement layer. Emitting unhealthy event",
					"startHeight", batch.StartHeight, "endHeight", batch.EndHeight, "error", err)
				heatlhEventData := &settlement.EventDataSettlementHealthStatus{Healthy: false, Error: err}
				utils.SubmitEventOrPanic(d.ctx, d.pubsub, heatlhEventData,
					map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}})
				// Sleep to allow context cancellation to take effect before retrying

				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Batch was submitted successfully. Wait for it to be accepted by the settlement layer.
		ticker := time.NewTicker(d.batchAcceptanceTimeout)
		defer ticker.Stop()

		select {
		case <-d.ctx.Done():
			return
		case <-subscription.Cancelled():
			d.logger.Debug("SLBatchPost subscription canceled")
			return
		case <-subscription.Out():
			d.logger.Info("Batch accepted by settlement layer. Emitting healthy event",
				"startHeight", batch.StartHeight, "endHeight", batch.EndHeight)
			heatlhEventData := &settlement.EventDataSettlementHealthStatus{Healthy: true}
			utils.SubmitEventOrPanic(d.ctx, d.pubsub, heatlhEventData,
				map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}})
			return
		case <-ticker.C:
			// Before emitting unhealthy event, check if the batch was accepted by the settlement layer and
			// we've just missed the event.
			includedBatch, err := d.waitForBatchInclusion(batch.StartHeight)
			if err != nil {
				// Batch was not accepted by the settlement layer. Emitting unhealthy event
				d.logger.Error("Batch not accepted by settlement layer. Emitting unhealthy event",
					"startHeight", batch.StartHeight, "endHeight", batch.EndHeight)
				heatlhEventData := &settlement.EventDataSettlementHealthStatus{Healthy: false, Error: settlement.ErrBatchNotAccepted}
				utils.SubmitEventOrPanic(d.ctx, d.pubsub, heatlhEventData,
					map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}})
				// Stop the ticker and restart the loop
				ticker.Stop()
				continue
			}

			d.logger.Info("Batch accepted by settlement layer", "startHeight", includedBatch.StartHeight, "endHeight", includedBatch.EndHeight)
			// Emit batch accepted event
			batchAcceptedEvent := &settlement.EventDataNewSettlementBatchAccepted{
				EndHeight:  includedBatch.EndHeight,
				StateIndex: includedBatch.StateIndex,
			}
			utils.SubmitEventOrPanic(d.ctx, d.pubsub, batchAcceptedEvent,
				map[string][]string{settlement.EventTypeKey: {settlement.EventNewSettlementBatchAccepted}})
			// Emit health event
			heatlhEventData := &settlement.EventDataSettlementHealthStatus{Healthy: true}
			utils.SubmitEventOrPanic(d.ctx, d.pubsub, heatlhEventData,
				map[string][]string{settlement.EventTypeKey: {settlement.EventSettlementHealthStatus}})
			return
		}
	}
}

// GetLatestBatch returns the latest batch from the Dymension Hub.
func (d *HubClient) GetLatestBatch(rollappID string) (*settlement.ResultRetrieveBatch, error) {
	latestStateInfoIndexResp, err := d.rollappQueryClient.LatestStateIndex(d.ctx,
		&rollapptypes.QueryGetLatestStateIndexRequest{RollappId: d.config.RollappID})
	if latestStateInfoIndexResp == nil {
		return nil, settlement.ErrBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	latestBatch, err := d.GetBatchAtIndex(rollappID, latestStateInfoIndexResp.StateIndex.Index)
	if err != nil {
		return nil, err
	}
	return latestBatch, nil
}

// GetBatchAtIndex returns the batch at the given index from the Dymension Hub.
func (d *HubClient) GetBatchAtIndex(rollappID string, index uint64) (*settlement.ResultRetrieveBatch, error) {
	stateInfoResp, err := d.rollappQueryClient.StateInfo(d.ctx,
		&rollapptypes.QueryGetStateInfoRequest{RollappId: d.config.RollappID, Index: index})
	if stateInfoResp == nil {
		return nil, settlement.ErrBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	return d.convertStateInfoToResultRetrieveBatch(&stateInfoResp.StateInfo)
}

// GetSequencers returns the sequence of the given rollapp.
func (d *HubClient) GetSequencers(rollappID string) ([]*types.Sequencer, error) {
	sequencers, err := d.sequencerQueryClient.SequencersByRollapp(d.ctx, &sequencertypes.QueryGetSequencersByRollappRequest{RollappId: d.config.RollappID})
	if err != nil {
		return nil, errors.Wrapf(settlement.ErrNoSequencerForRollapp, "rollappID: %s", rollappID)
	}
	sequencersList := []*types.Sequencer{}
	for _, sequencer := range sequencers.SequencerInfoList {
		var pubKey cryptotypes.PubKey
		err := d.protoCodec.UnpackAny(sequencer.Sequencer.DymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}
		var status types.SequencerStatus
		if sequencer.Status == sequencertypes.Proposer {
			status = types.Proposer
		} else {
			status = types.Inactive
		}
		if err != nil {
			return nil, err
		}
		sequencersList = append(sequencersList, &types.Sequencer{
			PublicKey: pubKey,
			Status:    status,
		})
	}
	return sequencersList, nil
}

func (d *HubClient) submitBatch(msgUpdateState *rollapptypes.MsgUpdateState) error {
	err := retry.Do(func() error {
		txResp, err := d.client.BroadcastTx(d.config.DymAccountName, msgUpdateState)
		if err != nil || txResp.Code != 0 {
			d.logger.Error("Error sending batch to settlement layer", "error", err)
			return err
		}
		return nil
	}, retry.Context(d.ctx), retry.LastErrorOnly(true), retry.Delay(d.batchRetryDelay),
		retry.MaxDelay(batchRetryMaxDelay), retry.Attempts(d.batchRetryAttempts))
	return err
}

func (d *HubClient) eventHandler() {
	// TODO(omritoptix): eventsChannel should be a generic channel which is later filtered by the event type.
	eventsChannel, err := d.client.SubscribeToEvents(d.ctx, "dymension-client", fmt.Sprintf(eventStateUpdate, d.config.RollappID))
	if err != nil {
		panic("Error subscribing to events")
	}
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.client.EventListenerQuit():
			// TODO(omritoptix): Fallback to polling
			panic("Settlement WS disconnected")
		case event := <-eventsChannel:
			// Assert value is in map and publish it to the event bus
			d.logger.Debug("Received event from settlement layer")
			_, ok := d.eventMap[event.Query]
			if !ok {
				d.logger.Debug("Ignoring event. Type not supported", "event", event)
				continue
			}
			eventData, err := d.getEventData(d.eventMap[event.Query], event)
			if err != nil {
				panic(err)
			}
			err = d.pubsub.PublishWithEvents(d.ctx, eventData, map[string][]string{settlement.EventTypeKey: {d.eventMap[event.Query]}})
			if err != nil {
				panic(err)
			}
		}
	}
}

func (d *HubClient) convertBatchToMsgUpdateState(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) (*rollapptypes.MsgUpdateState, error) {
	account, err := d.client.GetAccount(d.config.DymAccountName)
	if err != nil {
		return nil, err
	}

	addr, err := account.Address(addressPrefix)
	if err != nil {
		return nil, err
	}

	DAMetaData := &settlement.DAMetaData{
		Height: daResult.DAHeight,
		Client: daClient,
	}
	blockDescriptors := make([]rollapptypes.BlockDescriptor, len(batch.Blocks))
	for index, block := range batch.Blocks {
		blockDescriptor := rollapptypes.BlockDescriptor{
			Height:    block.Header.Height,
			StateRoot: block.Header.AppHash[:],
			// TODO(omritoptix): Change to a real ISR once supported
			IntermediateStatesRoot: make([]byte, 32),
		}
		blockDescriptors[index] = blockDescriptor
	}
	settlementBatch := &rollapptypes.MsgUpdateState{
		Creator:     addr,
		RollappId:   d.config.RollappID,
		StartHeight: batch.StartHeight,
		NumBlocks:   batch.EndHeight - batch.StartHeight + 1,
		DAPath:      DAMetaData.ToPath(),
		Version:     dymRollappVersion,
		BDs:         rollapptypes.BlockDescriptors{BD: blockDescriptors},
	}
	return settlementBatch, nil

}

func getCosmosClientOptions(config *settlement.Config) []cosmosclient.Option {
	if config.GasLimit == 0 {
		config.GasLimit = defaultGasLimit
	}
	options := []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithBroadcastMode(flags.BroadcastSync),
		cosmosclient.WithNodeAddress(config.NodeAddress),
		cosmosclient.WithFees(config.GasFees),
		cosmosclient.WithGasLimit(config.GasLimit),
		cosmosclient.WithGasPrices(config.GasPrices),
	}
	if config.KeyringHomeDir != "" {
		options = append(options,
			cosmosclient.WithKeyringBackend(cosmosaccount.KeyringBackend(config.KeyringBackend)),
			cosmosclient.WithHome(config.KeyringHomeDir),
		)
	}
	return options
}

func (d *HubClient) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case settlement.EventNewSettlementBatchAccepted:
		return d.convertToNewBatchEvent(rawEventData)
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

func (d *HubClient) convertToNewBatchEvent(rawEventData ctypes.ResultEvent) (*settlement.EventDataNewSettlementBatchAccepted, error) {
	var multiErr *multierror.Error
	numBlocks, err := strconv.ParseInt(rawEventData.Events["state_update.num_blocks"][0], 10, 64)
	multiErr = multierror.Append(multiErr, err)
	startHeight, err := strconv.ParseInt(rawEventData.Events["state_update.start_height"][0], 10, 64)
	multiErr = multierror.Append(multiErr, err)
	stateIndex, err := strconv.ParseInt(rawEventData.Events["state_update.state_info_index"][0], 10, 64)
	multiErr = multierror.Append(multiErr, err)
	err = multiErr.ErrorOrNil()
	if err != nil {
		return nil, multiErr
	}
	endHeight := uint64(startHeight + numBlocks - 1)
	NewBatchEvent := &settlement.EventDataNewSettlementBatchAccepted{
		EndHeight:  endHeight,
		StateIndex: uint64(stateIndex),
	}
	return NewBatchEvent, nil
}

func (d *HubClient) convertStateInfoToResultRetrieveBatch(stateInfo *rollapptypes.StateInfo) (*settlement.ResultRetrieveBatch, error) {
	daMetaData := &settlement.DAMetaData{}
	daMetaData, err := daMetaData.FromPath(stateInfo.DAPath)
	if err != nil {
		return nil, err
	}
	batchResult := &settlement.Batch{
		StartHeight: stateInfo.StartHeight,
		EndHeight:   stateInfo.StartHeight + stateInfo.NumBlocks - 1,
		MetaData: &settlement.BatchMetaData{
			DA: daMetaData,
		},
	}
	return &settlement.ResultRetrieveBatch{
		BaseResult: settlement.BaseResult{Code: settlement.StatusSuccess, StateIndex: stateInfo.StateInfoIndex.Index},
		Batch:      batchResult}, nil
}

// TODO(omritoptix): Change the retry attempts to be only for the batch polling. Also we need to have a more
// bullet proof check as theoretically the tx can stay in the mempool longer then our retry attempts.
func (d *HubClient) waitForBatchInclusion(batchStartHeight uint64) (*settlement.ResultRetrieveBatch, error) {
	var resultRetriveBatch *settlement.ResultRetrieveBatch
	err := retry.Do(func() error {
		latestBatch, err := d.GetLatestBatch(d.config.RollappID)
		if err != nil {
			return err
		}
		if latestBatch.Batch.StartHeight == batchStartHeight {
			resultRetriveBatch = latestBatch
			return nil
		}
		return settlement.ErrBatchNotFound
	}, retry.Context(d.ctx), retry.LastErrorOnly(true),
		retry.Delay(d.batchRetryDelay), retry.Attempts(d.batchRetryAttempts), retry.MaxDelay(batchRetryMaxDelay))
	return resultRetriveBatch, err
}
