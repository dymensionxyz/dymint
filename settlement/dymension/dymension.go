package dymension

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
	sequencertypes "github.com/dymensionxyz/dymension/v3/x/sequencer/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/pubsub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
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

// Client is the client for the Dymension Hub.
type Client struct {
	config                  *settlement.Config
	logger                  types.Logger
	pubsub                  *pubsub.Server
	cosmosClient            CosmosClient
	ctx                     context.Context
	cancel                  context.CancelFunc
	rollappQueryClient      rollapptypes.QueryClient
	sequencerQueryClient    sequencertypes.QueryClient
	protoCodec              *codec.ProtoCodec
	eventMap                map[string]string
	sequencerList           []*types.Sequencer
	retryAttempts           uint
	retryMinDelay           time.Duration
	retryMaxDelay           time.Duration
	batchAcceptanceTimeout  time.Duration
	batchAcceptanceAttempts uint
}

var _ settlement.ClientI = &Client{}

// Init is called once. it initializes the struct members.
func (c *Client) Init(config settlement.Config, pubsub *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
	ctx, cancel := context.WithCancel(context.Background())
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, config.RollappID):          settlement.EventNewBatchAccepted,
		fmt.Sprintf(eventSequencersListUpdate, config.RollappID): settlement.EventSequencersListUpdated,
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	c.config = &config
	c.logger = logger
	c.pubsub = pubsub
	c.ctx = ctx
	c.cancel = cancel
	c.eventMap = eventMap
	c.protoCodec = protoCodec
	c.retryAttempts = config.RetryAttempts
	c.batchAcceptanceTimeout = config.BatchAcceptanceTimeout
	c.batchAcceptanceAttempts = config.BatchAcceptanceAttempts
	c.retryMinDelay = config.RetryMinDelay
	c.retryMaxDelay = config.RetryMaxDelay

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	if c.cosmosClient == nil {
		client, err := cosmosclient.New(
			ctx,
			getCosmosClientOptions(&config)...,
		)
		if err != nil {
			return err
		}
		c.cosmosClient = NewCosmosClient(client)
	}
	c.rollappQueryClient = c.cosmosClient.GetRollappClient()
	c.sequencerQueryClient = c.cosmosClient.GetSequencerClient()

	return nil
}

// Start starts the HubClient.
func (c *Client) Start() error {
	err := c.cosmosClient.StartEventListener()
	if err != nil {
		return err
	}
	go c.eventHandler()
	return nil
}

// Stop stops the HubClient.
func (c *Client) Stop() error {
	c.cancel()
	err := c.cosmosClient.StopEventListener()
	if err != nil {
		return err
	}

	return nil
}

// SubmitBatch posts a batch to the Dymension Hub. it tries to post the batch until it is accepted by the settlement layer.
// it emits success and failure events to the event bus accordingly.
func (c *Client) SubmitBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) error {
	msgUpdateState, err := c.convertBatchToMsgUpdateState(batch, daResult)
	if err != nil {
		return fmt.Errorf("convert batch to msg update state: %w", err)
	}

	// TODO: probably should be changed to be a channel, as the eventHandler is also in the HubClient in he produces the event
	postBatchSubscriberClient := fmt.Sprintf("%s-%d-%s", postBatchSubscriberPrefix, batch.StartHeight(), uuid.New().String())
	subscription, err := c.pubsub.Subscribe(c.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted, 1000)
	if err != nil {
		return fmt.Errorf("pub sub subscribe to settlement state updates: %w", err)
	}

	//nolint:errcheck
	defer c.pubsub.Unsubscribe(c.ctx, postBatchSubscriberClient, settlement.EventQueryNewSettlementBatchAccepted)

	for {
		// broadcast loop: broadcast the transaction to the blockchain (with infinite retries).
		err := c.RunWithRetryInfinitely(func() error {
			err := c.broadcastBatch(msgUpdateState)
			if err != nil {
				if errors.Is(err, gerrc.ErrAlreadyExists) {
					return retry.Unrecoverable(err)
				}

				c.logger.Error(
					"Submit batch",
					"startHeight",
					batch.StartHeight(),
					"endHeight",
					batch.EndHeight,
					"error",
					err,
				)
			}
			return err
		})
		if err != nil {
			// this could happen if we timed-out waiting for acceptance in the previous iteration, but the batch was indeed submitted
			if errors.Is(err, gerrc.ErrAlreadyExists) {
				c.logger.Debug("Batch already accepted", "startHeight", batch.StartHeight(), "endHeight", batch.EndHeight())
				return nil
			}
			return fmt.Errorf("broadcast batch: %w", err)
		}

		// Batch was submitted successfully. Wait for it to be accepted by the settlement layer.
		timer := time.NewTimer(c.batchAcceptanceTimeout)
		defer timer.Stop()
		attempt := uint64(1)

		for {
			select {
			case <-c.ctx.Done():
				return c.ctx.Err()

			case <-subscription.Cancelled():
				return fmt.Errorf("subscription cancelled")

			case event := <-subscription.Out():
				eventData, _ := event.Data().(*settlement.EventDataNewBatchAccepted)
				if eventData.EndHeight != batch.EndHeight() {
					c.logger.Debug("Received event for a different batch, ignoring.", "event", eventData)
					continue // continue waiting for acceptance of the current batch
				}
				c.logger.Info("Batch accepted.", "startHeight", batch.StartHeight(), "endHeight", batch.EndHeight(), "stateIndex", eventData.StateIndex)
				return nil

			case <-timer.C:
				// Check if the batch was accepted by the settlement layer, and we've just missed the event.
				includedBatch, err := c.pollForBatchInclusion(batch.EndHeight())
				timer.Reset(c.batchAcceptanceTimeout)
				// no error, but still not included
				if err == nil && !includedBatch {
					attempt++
					if attempt <= uint64(c.batchAcceptanceAttempts) {
						continue // continue waiting for acceptance of the current batch
					}
					c.logger.Error(
						"Timed out waiting for batch inclusion on settlement layer",
						"startHeight",
						batch.StartHeight(),
						"endHeight",
						batch.EndHeight(),
					)
					break // breaks the switch case, and goes back to the broadcast loop
				}
				if err != nil {
					c.logger.Error(
						"Wait for batch inclusion",
						"startHeight",
						batch.StartHeight(),
						"endHeight",
						batch.EndHeight(),
						"error",
						err,
					)
					continue // continue waiting for acceptance of the current batch
				}
				// all good
				c.logger.Info("Batch accepted", "startHeight", batch.StartHeight(), "endHeight", batch.EndHeight())
				return nil
			}
			break // failed waiting for acceptance. broadcast the batch again
		}
	}
}

func (c *Client) getStateInfo(index, height *uint64) (res *rollapptypes.QueryGetStateInfoResponse, err error) {
	req := &rollapptypes.QueryGetStateInfoRequest{RollappId: c.config.RollappID}
	if index != nil {
		req.Index = *index
	}
	if height != nil {
		req.Height = *height
	}
	err = c.RunWithRetry(func() error {
		res, err = c.rollappQueryClient.StateInfo(c.ctx, req)

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(gerrc.ErrNotFound)
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("query state info: %w: %w", gerrc.ErrUnknown, err)
	}
	if res == nil { // not supposed to happen
		return nil, fmt.Errorf("empty response with nil err: %w", gerrc.ErrUnknown)
	}
	return
}

// GetLatestBatch returns the latest batch from the Dymension Hub.
func (c *Client) GetLatestBatch() (*settlement.ResultRetrieveBatch, error) {
	res, err := c.getStateInfo(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

// GetBatchAtIndex returns the batch at the given index from the Dymension Hub.
func (c *Client) GetBatchAtIndex(index uint64) (*settlement.ResultRetrieveBatch, error) {
	res, err := c.getStateInfo(&index, nil)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

func (c *Client) GetHeightState(h uint64) (*settlement.ResultGetHeightState, error) {
	res, err := c.getStateInfo(nil, &h)
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

// GetProposer implements settlement.ClientI.
func (c *Client) GetProposer() *types.Sequencer {
	seqs, err := c.GetSequencers()
	if err != nil {
		c.logger.Error("Get sequencers", "error", err)
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
func (c *Client) GetSequencers() ([]*types.Sequencer, error) {
	if c.sequencerList != nil {
		return c.sequencerList, nil
	}

	var res *sequencertypes.QueryGetSequencersByRollappByStatusResponse
	req := &sequencertypes.QueryGetSequencersByRollappByStatusRequest{
		RollappId: c.config.RollappID,
		Status:    sequencertypes.Bonded,
	}
	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.sequencerQueryClient.SequencersByRollappByStatus(c.ctx, req)
		return err
	})
	if err != nil {
		return nil, err
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	sequencersList := make([]*types.Sequencer, 0, len(res.Sequencers))
	for _, sequencer := range res.Sequencers {
		var pubKey cryptotypes.PubKey
		err := c.protoCodec.UnpackAny(sequencer.DymintPubKey, &pubKey)
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
	c.sequencerList = sequencersList
	return sequencersList, nil
}

func (c *Client) broadcastBatch(msgUpdateState *rollapptypes.MsgUpdateState) error {
	txResp, err := c.cosmosClient.BroadcastTx(c.config.DymAccountName, msgUpdateState)
	if err != nil {
		if strings.Contains(err.Error(), rollapptypes.ErrWrongBlockHeight.Error()) {
			err = fmt.Errorf("%w: %w", err, gerrc.ErrAlreadyExists)
		}
		return fmt.Errorf("broadcast tx: %w", err)
	}
	if txResp.Code != 0 {
		return fmt.Errorf("broadcast tx status code is not 0: %w", gerrc.ErrUnknown)
	}
	return nil
}

func (c *Client) eventHandler() {
	// TODO(omritoptix): eventsChannel should be a generic channel which is later filtered by the event type.
	subscriber := fmt.Sprintf("dymension-client-%s", uuid.New().String())
	eventsChannel, err := c.cosmosClient.SubscribeToEvents(c.ctx, subscriber, fmt.Sprintf(eventStateUpdate, c.config.RollappID), 1000)
	if err != nil {
		panic("Error subscribing to events")
	}
	// TODO: add defer unsubscribeAll

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.cosmosClient.EventListenerQuit():
			// TODO(omritoptix): Fallback to polling
			panic("Settlement WS disconnected")
		case event := <-eventsChannel:
			// Assert value is in map and publish it to the event bus
			c.logger.Info("new sl event", "event", event)
			data, ok := c.eventMap[event.Query]
			if !ok {
				c.logger.Debug("Ignoring event. Type not supported", "event", event)
				continue
			}
			eventData, err := c.getEventData(data, event)
			if err != nil {
				panic(err)
			}
			uevent.MustPublish(c.ctx, c.pubsub, eventData, map[string][]string{settlement.EventTypeKey: {data}})
		}
	}
}

func (c *Client) convertBatchToMsgUpdateState(batch *types.Batch, daResult *da.ResultSubmitBatch) (*rollapptypes.MsgUpdateState, error) {
	account, err := c.cosmosClient.GetAccount(c.config.DymAccountName)
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
		RollappId:   c.config.RollappID,
		StartHeight: batch.StartHeight(),
		NumBlocks:   batch.NumBlocks(),
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

func (c *Client) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case settlement.EventNewBatchAccepted:
		return convertToNewBatchEvent(rawEventData)
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

// pollForBatchInclusion polls the hub for the inclusion of a batch with the given end height.
func (c *Client) pollForBatchInclusion(batchEndHeight uint64) (bool, error) {
	latestBatch, err := c.GetLatestBatch()
	if err != nil {
		return false, fmt.Errorf("get latest batch: %w", err)
	}

	return latestBatch.Batch.EndHeight == batchEndHeight, nil
}
