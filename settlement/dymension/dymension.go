package dymension

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/dymensionxyz/dymint/gerr"

	uevent "github.com/dymensionxyz/dymint/utils/event"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	"github.com/google/uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sequencertypes "github.com/dymensionxyz/dymension/v3/x/sequencer/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
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
	eventStateUpdate          = "state_update.rollapp_id='%s' AND state_update.status='PENDING'"
	eventSequencersListUpdate = "sequencers_list_update.rollapp_id='%s'"
)

const (
	postBatchSubscriberPrefix = "postBatchSubscriber"
)

// LayerClient is the client for the Dymension Hub.
type LayerClient struct {
	config               *settlement.Config
	logger               types.Logger
	pubsub               *pubsub.Server
	cosmosClient         CosmosClient
	ctx                  context.Context
	cancel               context.CancelFunc
	rollappQueryClient   rollapptypes.QueryClient
	sequencerQueryClient sequencertypes.QueryClient
	protoCodec           *codec.ProtoCodec
	eventMap             map[string]string
	// channel for getting notified when a batch is accepted by the settlement layer.
	// only one batch of a specific height can get accepted and we can are currently sending only one batch at a time.
	// for that reason it's safe to assume that if a batch is accepted, it refers to the last batch we've sent.
	retryAttempts          uint
	retryMinDelay          time.Duration
	retryMaxDelay          time.Duration
	batchAcceptanceTimeout time.Duration
}

var _ settlement.LayerI = &LayerClient{}

func NewDymensionHubClient(config settlement.Config, pubsub *pubsub.Server, logger types.Logger, options ...Option) (*LayerClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, config.RollappID):          settlement.EventNewBatchAccepted,
		fmt.Sprintf(eventSequencersListUpdate, config.RollappID): settlement.EventSequencersListUpdated,
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	dymesionHubClient := &LayerClient{
		config:                 &config,
		logger:                 logger,
		pubsub:                 pubsub,
		ctx:                    ctx,
		cancel:                 cancel,
		eventMap:               eventMap,
		protoCodec:             protoCodec,
		retryAttempts:          config.RetryAttempts,
		batchAcceptanceTimeout: config.BatchAcceptanceTimeout,
		retryMinDelay:          config.RetryMinDelay,
		retryMaxDelay:          config.RetryMaxDelay,
	}

	for _, option := range options {
		option(dymesionHubClient)
	}

	if dymesionHubClient.cosmosClient == nil {
		client, err := cosmosclient.New(
			ctx,
			getCosmosClientOptions(&config)...,
		)
		if err != nil {
			return nil, err
		}
		dymesionHubClient.cosmosClient = NewCosmosClient(client)
	}
	dymesionHubClient.rollappQueryClient = dymesionHubClient.cosmosClient.GetRollappClient()
	dymesionHubClient.sequencerQueryClient = dymesionHubClient.cosmosClient.GetSequencerClient()

	return dymesionHubClient, nil
}

// Init is called once. it initializes the struct members.
func (dlc *LayerClient) Init(config settlement.Config, pubsub *pubsub.Server, logger types.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, config.RollappID):          settlement.EventNewBatchAccepted,
		fmt.Sprintf(eventSequencersListUpdate, config.RollappID): settlement.EventSequencersListUpdated,
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	dlc.config = &config
	dlc.logger = logger
	dlc.pubsub = pubsub
	dlc.ctx = ctx
	dlc.cancel = cancel
	dlc.eventMap = eventMap
	dlc.protoCodec = protoCodec
	dlc.retryAttempts = config.RetryAttempts
	dlc.batchAcceptanceTimeout = config.BatchAcceptanceTimeout
	dlc.retryMinDelay = config.RetryMinDelay
	dlc.retryMaxDelay = config.RetryMaxDelay

	if dlc.cosmosClient == nil {
		client, err := cosmosclient.New(
			ctx,
			getCosmosClientOptions(&config)...,
		)
		if err != nil {
			return err
		}
		dlc.cosmosClient = NewCosmosClient(client)
	}
	dlc.rollappQueryClient = dlc.cosmosClient.GetRollappClient()
	dlc.sequencerQueryClient = dlc.cosmosClient.GetSequencerClient()

	return nil
}

// Start starts the HubClient.
func (d *LayerClient) Start() error {
	err := d.cosmosClient.StartEventListener()
	if err != nil {
		return err
	}
	go d.eventHandler()
	return nil
}

// Stop stops the HubClient.
func (d *LayerClient) Stop() error {
	d.cancel()
	err := d.cosmosClient.StopEventListener()
	if err != nil {
		return err
	}

	return nil
}

// SubmitBatch posts a batch to the Dymension Hub. it tries to post the batch until it is accepted by the settlement layer.
// it emits success and failure events to the event bus accordingly.
func (d *LayerClient) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	msgUpdateState, err := d.convertBatchToMsgUpdateState(batch, daResult)
	if err != nil {
		return fmt.Errorf("convert batch to msg update state: %w", err)
	}

	// TODO: probably should be changed to be a channel, as the eventHandler is also in the HubClient in he produces the event
	postBatchSubscriberClient := fmt.Sprintf("%s-%d-%s", postBatchSubscriberPrefix, batch.StartHeight, uuid.New().String())
	subscription, err := d.pubsub.Subscribe(d.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		return fmt.Errorf("pub sub subscribe to settlement state updates: %w", err)
	}

	//nolint:errcheck
	defer d.pubsub.Unsubscribe(d.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted)

	// Try submitting the batch to the settlement layer. If submission (i.e. only submission, not acceptance) fails we emit an unhealthy event
	// and try again in the next loop. If submission succeeds we wait for the batch to be accepted by the settlement layer.
	// If it is not accepted we emit an unhealthy event and start again the submission loop.
	for {
		select {
		case <-d.ctx.Done():
			return d.ctx.Err()
		default:
			err := d.submitBatch(msgUpdateState)
			if err != nil {
				d.logger.Error(
					"Submit batch",
					"startHeight",
					batch.StartHeight,
					"endHeight",
					batch.EndHeight,
					"error",
					err,
				)

				// Sleep to allow context cancellation to take effect before retrying
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		// Batch was submitted successfully. Wait for it to be accepted by the settlement layer.
		timer := time.NewTimer(d.batchAcceptanceTimeout)
		defer timer.Stop()

		select {
		case <-d.ctx.Done():
			return d.ctx.Err()

		case <-subscription.Cancelled():
			return fmt.Errorf("subscription cancelled")

		case <-subscription.Out():
			d.logger.Debug("Batch accepted", "startHeight", batch.StartHeight, "endHeight", batch.EndHeight)
			return nil

		case <-timer.C:
			// Check if the batch was accepted by the settlement layer, and we've just missed the event.
			includedBatch, err := d.waitForBatchInclusion(batch.StartHeight)
			if err != nil {
				d.logger.Error(
					"Wait for batch inclusion",
					"startHeight",
					batch.StartHeight,
					"endHeight",
					batch.EndHeight,
					"error",
					err,
				)

				timer.Stop() // we don't forget to clean up
				continue
			}

			// all good
			d.logger.Info("Batch accepted", "startHeight", includedBatch.StartHeight, "endHeight", includedBatch.EndHeight)
			return nil
		}
	}
}

func (d *LayerClient) getStateInfo(index, height *uint64) (res *rollapptypes.QueryGetStateInfoResponse, err error) {
	req := &rollapptypes.QueryGetStateInfoRequest{RollappId: d.config.RollappID}
	if index != nil {
		req.Index = *index
	}
	if height != nil {
		req.Height = *height
	}
	err = d.RunWithRetry(func() error {
		res, err = d.rollappQueryClient.StateInfo(d.ctx, req)

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(gerr.ErrNotFound)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("query state info: %w: %w", gerr.ErrUnknown, err)
	}
	if res == nil { // not supposed to happen
		return nil, fmt.Errorf("empty response with nil err: %w", gerr.ErrUnknown)
	}
	return
}

// RetrieveBatch implements settlement.LayerI.
func (dlc *LayerClient) RetrieveBatch(stateIndex ...uint64) (*settlement.ResultRetrieveBatch, error) {
	if len(stateIndex) == 0 {
		dlc.logger.Debug("Getting latest batch from settlement layer")
		return dlc.GetLatestBatch()
	}
	if len(stateIndex) == 1 {
		dlc.logger.Debug("Getting batch from settlement layer", "state index", stateIndex)
		return dlc.GetBatchAtIndex(stateIndex[0])
	}
	return nil, fmt.Errorf("expected 0 or 1 index: got %d: %w", len(stateIndex), gerr.ErrInvalidArgument)
}

// GetLatestBatch returns the latest batch from the Dymension Hub.
func (d *LayerClient) GetLatestBatch() (*settlement.ResultRetrieveBatch, error) {
	res, err := d.getStateInfo(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return d.convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

// GetBatchAtIndex returns the batch at the given index from the Dymension Hub.
func (d *LayerClient) GetBatchAtIndex(index uint64) (*settlement.ResultRetrieveBatch, error) {
	res, err := d.getStateInfo(&index, nil)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return d.convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

func (d *LayerClient) GetHeightState(h uint64) (*settlement.ResultGetHeightState, error) {
	res, err := d.getStateInfo(nil, &h)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return &settlement.ResultGetHeightState{
		ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess},
		State: settlement.State{
			StateIndex: res.GetStateInfo().StateInfoIndex.Index,
		},
	}, nil
}

// GetProposer implements settlement.LayerI.
func (dlc *LayerClient) GetProposer() *types.Sequencer {
	seqs, err := dlc.GetSequencers()
	if err != nil {
		dlc.logger.Error("Get sequencers", "error", err)
		return nil
	}
	for _, sequencer := range seqs {
		if sequencer.Status == types.Proposer {
			return sequencer
		}
	}
	return nil
}

// GetSequencers returns the bonded sequencers of the given rollapp.
// FIXME: add caching
func (d *LayerClient) GetSequencers() ([]*types.Sequencer, error) {
	var res *sequencertypes.QueryGetSequencersByRollappByStatusResponse
	req := &sequencertypes.QueryGetSequencersByRollappByStatusRequest{
		RollappId: d.config.RollappID,
		Status:    sequencertypes.Bonded,
	}
	err := d.RunWithRetry(func() error {
		var err error
		res, err = d.sequencerQueryClient.SequencersByRollappByStatus(d.ctx, req)
		return err
	})
	if err != nil {
		return nil, err
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerr.ErrUnknown)
	}

	sequencersList := make([]*types.Sequencer, 0, len(res.Sequencers))
	for _, sequencer := range res.Sequencers {
		var pubKey cryptotypes.PubKey
		err := d.protoCodec.UnpackAny(sequencer.DymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}

		status := types.Inactive
		if sequencer.Proposer {
			status = types.Proposer
		}

		sequencersList = append(sequencersList, &types.Sequencer{
			PublicKey: pubKey,
			Status:    status,
		})
	}
	return sequencersList, nil
}

func (d *LayerClient) submitBatch(msgUpdateState *rollapptypes.MsgUpdateState) error {
	err := d.RunWithRetry(func() error {
		txResp, err := d.cosmosClient.BroadcastTx(d.config.DymAccountName, msgUpdateState)
		if err != nil || txResp.Code != 0 {
			return fmt.Errorf("broadcast tx: %w", err)
		}
		return nil
	})
	return err
}

func (d *LayerClient) eventHandler() {
	// TODO(omritoptix): eventsChannel should be a generic channel which is later filtered by the event type.
	eventsChannel, err := d.cosmosClient.SubscribeToEvents(d.ctx, "dymension-client", fmt.Sprintf(eventStateUpdate, d.config.RollappID))
	if err != nil {
		panic("Error subscribing to events")
	}
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.cosmosClient.EventListenerQuit():
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
			uevent.MustPublish(d.ctx, d.pubsub, eventData, map[string][]string{settlement.EventTypeKey: {d.eventMap[event.Query]}})
		}
	}
}

func (d *LayerClient) convertBatchToMsgUpdateState(batch *types.Batch, daResult *da.ResultSubmitBatch) (*rollapptypes.MsgUpdateState, error) {
	account, err := d.cosmosClient.GetAccount(d.config.DymAccountName)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	addr, err := account.Address(addressPrefix)
	if err != nil {
		return nil, fmt.Errorf("derive address: %w", err)
	}

	blockDescriptors := make([]rollapptypes.BlockDescriptor, len(batch.Blocks))
	for index, block := range batch.Blocks {
		blockDescriptor := rollapptypes.BlockDescriptor{
			Height:    block.Header.Height,
			StateRoot: block.Header.AppHash[:],
		}
		blockDescriptors[index] = blockDescriptor
	}

	settlementBatch := &rollapptypes.MsgUpdateState{
		Creator:     addr,
		RollappId:   d.config.RollappID,
		StartHeight: batch.StartHeight,
		NumBlocks:   batch.EndHeight - batch.StartHeight + 1,
		DAPath:      daResult.SubmitMetaData.ToPath(),
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

func (d *LayerClient) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case settlement.EventNewBatchAccepted:
		return d.convertToNewBatchEvent(rawEventData)
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

func (d *LayerClient) convertToNewBatchEvent(rawEventData ctypes.ResultEvent) (*settlement.EventDataNewBatchAccepted, error) {
	// check all expected attributes  exists
	events := rawEventData.Events
	if events["state_update.num_blocks"] == nil || events["state_update.start_height"] == nil || events["state_update.state_info_index"] == nil {
		return nil, fmt.Errorf("missing expected attributes in event")
	}

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
	NewBatchEvent := &settlement.EventDataNewBatchAccepted{
		EndHeight:  endHeight,
		StateIndex: uint64(stateIndex),
	}
	return NewBatchEvent, nil
}

func (d *LayerClient) convertStateInfoToResultRetrieveBatch(stateInfo *rollapptypes.StateInfo) (*settlement.ResultRetrieveBatch, error) {
	daMetaData := &da.DASubmitMetaData{}
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
		ResultBase: settlement.ResultBase{Code: settlement.StatusSuccess, StateIndex: stateInfo.StateInfoIndex.Index},
		Batch:      batchResult,
	}, nil
}

// TODO(omritoptix): Change the retry attempts to be only for the batch polling. Also we need to have a more
// TODO: bullet proof check as theoretically the tx can stay in the mempool longer then our retry attempts.
func (d *LayerClient) waitForBatchInclusion(batchStartHeight uint64) (*settlement.ResultRetrieveBatch, error) {
	var res *settlement.ResultRetrieveBatch
	err := d.RunWithRetry(
		func() error {
			latestBatch, err := d.GetLatestBatch()
			if err != nil {
				return fmt.Errorf("get latest batch: %w", err)
			}
			if latestBatch.Batch.StartHeight != batchStartHeight {
				return fmt.Errorf("latest batch start height not match expected start height: %w", gerr.ErrNotFound)
			}
			res = latestBatch
			return nil
		},
	)
	return res, err
}

// RunWithRetry runs the given operation with retry, doing a number of attempts, and taking the last
// error only. It uses the context of the HubClient.
func (d *LayerClient) RunWithRetry(operation func() error) error {
	return retry.Do(operation,
		retry.Context(d.ctx),
		retry.LastErrorOnly(true),
		retry.Delay(d.retryMinDelay),
		retry.Attempts(d.retryAttempts),
		retry.MaxDelay(d.retryMaxDelay),
	)
}
