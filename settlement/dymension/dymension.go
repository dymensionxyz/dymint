package dymension

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymint/third_party/dymension/rollapp/types"
	sequencertypes "github.com/dymensionxyz/dymint/third_party/dymension/sequencer/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

const (
	addressPrefix = "dym"
)

const (
	postBatchSubscriberPrefix = "postBatchSubscriber"
)

// Client is the client for the Dymension Hub.
type Client struct {
	config                  *settlement.Config
	rollappId               string
	logger                  types.Logger
	pubsub                  *pubsub.Server
	cosmosClient            CosmosClient
	ctx                     context.Context
	rollappQueryClient      rollapptypes.QueryClient
	sequencerQueryClient    sequencertypes.QueryClient
	protoCodec              *codec.ProtoCodec
	proposer                types.Sequencer
	retryAttempts           uint
	retryMinDelay           time.Duration
	retryMaxDelay           time.Duration
	batchAcceptanceTimeout  time.Duration
	batchAcceptanceAttempts uint
}

var _ settlement.ClientI = &Client{}

// Init is called once. it initializes the struct members.
func (c *Client) Init(config settlement.Config, rollappId string, pubsub *pubsub.Server, logger types.Logger, options ...settlement.Option) error {
	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	c.rollappId = rollappId
	c.config = &config
	c.logger = logger
	c.pubsub = pubsub
	c.ctx = context.Background()
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
	return c.cosmosClient.StopEventListener()
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
	defer c.pubsub.UnsubscribeAll(c.ctx, postBatchSubscriberClient)

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
					batch.EndHeight(),
					"error",
					err,
				)
			}
			return err
		})
		if err != nil {
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
				c.logger.Info("Batch accepted.", "startHeight", batch.StartHeight(), "endHeight", batch.EndHeight(), "stateIndex", eventData.StateIndex, "dapath", msgUpdateState.DAPath)
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
	req := &rollapptypes.QueryGetStateInfoRequest{RollappId: c.rollappId}
	if index != nil {
		req.Index = *index
	}
	if height != nil {
		req.Height = *height
	}
	err = c.RunWithRetry(func() error {
		res, err = c.rollappQueryClient.StateInfo(c.ctx, req)

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("query state info: %w", err)
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
	// return cached proposer
	if !c.proposer.IsEmpty() {
		return &c.proposer
	}

	seqs, err := c.GetBondedSequencers()
	if err != nil {
		c.logger.Error("GetBondedSequencers", "error", err)
		return nil
	}

	var proposerAddr string
	err = c.RunWithRetry(func() error {
		reqProposer := &sequencertypes.QueryGetProposerByRollappRequest{
			RollappId: c.rollappId,
		}
		res, err := c.sequencerQueryClient.GetProposerByRollapp(c.ctx, reqProposer)
		if err == nil {
			proposerAddr = res.ProposerAddr
			return nil
		}
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	})
	if err != nil {
		c.logger.Error("GetProposer", "error", err)
		return nil
	}

	// find the sequencer with the proposer address
	index := slices.IndexFunc(seqs, func(seq types.Sequencer) bool {
		return seq.SettlementAddress == proposerAddr
	})
	// will return nil if the proposer is not set
	if index == -1 {
		return nil
	}
	c.proposer = seqs[index]
	return &seqs[index]
}

// GetAllSequencers returns all sequencers of the given rollapp.
func (c *Client) GetAllSequencers() ([]types.Sequencer, error) {
	var res *sequencertypes.QueryGetSequencersByRollappResponse
	req := &sequencertypes.QueryGetSequencersByRollappRequest{
		RollappId: c.rollappId,
	}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.sequencerQueryClient.SequencersByRollapp(c.ctx, req)
		if err == nil {
			return nil
		}

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	var sequencerList []types.Sequencer
	for _, sequencer := range res.Sequencers {
		var pubKey cryptotypes.PubKey
		err := c.protoCodec.UnpackAny(sequencer.DymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}

		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
		if err != nil {
			return nil, err
		}

		sequencerList = append(sequencerList, *types.NewSequencer(tmPubKey, sequencer.Address))
	}

	return sequencerList, nil
}

// GetBondedSequencers returns the bonded sequencers of the given rollapp.
func (c *Client) GetBondedSequencers() ([]types.Sequencer, error) {
	var res *sequencertypes.QueryGetSequencersByRollappByStatusResponse
	req := &sequencertypes.QueryGetSequencersByRollappByStatusRequest{
		RollappId: c.rollappId,
		Status:    sequencertypes.Bonded,
	}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.sequencerQueryClient.SequencersByRollappByStatus(c.ctx, req)
		if err == nil {
			return nil
		}

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	var sequencerList []types.Sequencer
	for _, sequencer := range res.Sequencers {
		var pubKey cryptotypes.PubKey
		err := c.protoCodec.UnpackAny(sequencer.DymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}

		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
		if err != nil {
			return nil, err
		}
		sequencerList = append(sequencerList, *types.NewSequencer(tmPubKey, sequencer.Address))
	}

	return sequencerList, nil
}

// CheckRotationInProgress implements settlement.ClientI.
func (c *Client) CheckRotationInProgress() (*types.Sequencer, error) {
	var (
		nextAddr string
		found    bool
	)
	err := c.RunWithRetry(func() error {
		req := &sequencertypes.QueryGetNextProposerByRollappRequest{
			RollappId: c.rollappId,
		}
		res, err := c.sequencerQueryClient.GetNextProposerByRollapp(c.ctx, req)
		if err == nil && res.RotationInProgress {
			nextAddr = res.NextProposerAddr
			found = true
			return nil
		}
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	if nextAddr == "" {
		return &types.Sequencer{}, nil
	}

	seqs, err := c.GetBondedSequencers()
	if err != nil {
		return nil, fmt.Errorf("get sequencers: %w", err)
	}

	for _, sequencer := range seqs {
		if sequencer.SettlementAddress == nextAddr {
			return &sequencer, nil
		}
	}

	return nil, fmt.Errorf("next proposer not found in bonded set: %w", gerrc.ErrInternal)
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

	c.logger.Info("Broadcasted batch", "txHash", txResp.TxHash)

	return nil
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
			Timestamp: block.Header.GetTimestamp(),
		}
		blockDescriptors[index] = blockDescriptor
	}

	settlementBatch := &rollapptypes.MsgUpdateState{
		Creator:     addr,
		RollappId:   c.rollappId,
		StartHeight: batch.StartHeight(),
		NumBlocks:   batch.NumBlocks(),
		DAPath:      daResult.SubmitMetaData.ToPath(),
		BDs:         rollapptypes.BlockDescriptors{BD: blockDescriptors},
		Last:        batch.LastBatch,
	}
	return settlementBatch, nil
}

func getCosmosClientOptions(config *settlement.Config) []cosmosclient.Option {
	var (
		gas           string
		gasAdjustment float64 = 1.0
	)
	if config.GasLimit == 0 {
		gas = "auto"
		gasAdjustment = 1.1
	} else {
		gas = strconv.FormatUint(config.GasLimit, 10)
	}
	options := []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithNodeAddress(config.NodeAddress),
		cosmosclient.WithFees(config.GasFees),
		cosmosclient.WithGas(gas),
		cosmosclient.WithGasAdjustment(gasAdjustment),
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

// pollForBatchInclusion polls the hub for the inclusion of a batch with the given end height.
func (c *Client) pollForBatchInclusion(batchEndHeight uint64) (bool, error) {
	latestBatch, err := c.GetLatestBatch()
	if err != nil {
		return false, fmt.Errorf("get latest batch: %w", err)
	}

	return latestBatch.Batch.EndHeight == batchEndHeight, nil
}
