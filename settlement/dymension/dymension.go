package dymension

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	querytypes "github.com/cosmos/cosmos-sdk/types/query"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/uuid"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
	sequencertypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/sequencer"
	protoutils "github.com/dymensionxyz/dymint/utils/proto"
)

const (
	addressPrefix     = "dym"
	SENTINEL_PROPOSER = "sentinel"
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
func (c *Client) SubmitBatch(batch *types.Batch, _ da.Client, daResult *da.ResultSubmitBatch) error {
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
				eventData, _ := event.Data().(*settlement.EventDataNewBatch)
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

func (c *Client) getStateInfo(index, height *uint64, tryRetry bool) (res *rollapptypes.QueryGetStateInfoResponse, err error) {
	req := &rollapptypes.QueryGetStateInfoRequest{
		RollappId: c.rollappId,
	}
	if index != nil {
		req.Index = *index
	}
	if height != nil {
		req.Height = *height
	}
	err = c.RunWithRetry(func() error {
		res, err = c.rollappQueryClient.StateInfo(c.ctx, req)
		if status.Code(err) != codes.NotFound {
			return err
		}
		if tryRetry {
			return errors.Join(gerrc.ErrNotFound, err)
		}
		return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
	})
	if err != nil {
		return nil, fmt.Errorf("query state info: %w", err)
	}
	if res == nil { // not supposed to happen
		return nil, fmt.Errorf("empty response with nil err: %w", gerrc.ErrUnknown)
	}
	return
}

func (c *Client) getLatestHeight(finalized bool) (res *rollapptypes.QueryGetLatestHeightResponse, err error) {
	req := &rollapptypes.QueryGetLatestHeightRequest{
		RollappId: c.rollappId,
		Finalized: finalized,
	}
	err = c.RunWithRetry(func() error {
		res, err = c.rollappQueryClient.LatestHeight(c.ctx, req)

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
	res, err := c.getStateInfo(nil, nil, false)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

// GetBatchAtIndex returns the batch at the given index from the Dymension Hub.
func (c *Client) GetBatchAtIndex(index uint64) (*settlement.ResultRetrieveBatch, error) {
	res, err := c.getStateInfo(&index, nil, true)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

// GetBatchAtHeight returns the batch at the given height from the Dymension Hub.
func (c *Client) GetBatchAtHeight(height uint64, retry ...bool) (*settlement.ResultRetrieveBatch, error) {
	tryRetry := true
	if len(retry) > 0 {
		tryRetry = retry[0]
	}
	res, err := c.getStateInfo(nil, &height, tryRetry)
	if err != nil {
		return nil, fmt.Errorf("get state info: %w", err)
	}
	return convertStateInfoToResultRetrieveBatch(&res.StateInfo)
}

// GetLatestHeight returns the latest state update height from the settlement layer.
func (c *Client) GetLatestHeight() (uint64, error) {
	res, err := c.getLatestHeight(false)
	if err != nil {
		return uint64(0), fmt.Errorf("get latest height: %w", err)
	}
	return res.Height, nil
}

// GetLatestFinalizedHeight returns the latest finalized height from the settlement layer.
func (c *Client) GetLatestFinalizedHeight() (uint64, error) {
	res, err := c.getLatestHeight(true)
	if err != nil {
		return uint64(0), fmt.Errorf("get latest height: %w", err)
	}
	return res.Height, nil
}

// GetProposerAtHeight return the proposer at height.
// In case of negative height, it will return the latest proposer.
func (c *Client) GetProposerAtHeight(height int64) (*types.Sequencer, error) {
	// Get all sequencers to find the proposer address
	seqs, err := c.GetAllSequencers()
	if err != nil {
		return nil, fmt.Errorf("get bonded sequencers: %w", err)
	}

	// Get either latest proposer or proposer at height
	var proposerAddr string
	if height < 0 {
		proposerAddr, err = c.getLatestProposer()
		if err != nil {
			return nil, fmt.Errorf("get latest proposer: %w", err)
		}
	} else {
		// Get the state info for the relevant height and get address from there
		res, err := c.GetBatchAtHeight(uint64(height), false)
		// if case of height not found, it may be because it didn't arrive to the hub yet.
		// In that case we want to return the current proposer.
		if err != nil {
			// If batch not found, fallback to latest proposer
			if errors.Is(err, gerrc.ErrNotFound) {
				proposerAddr, err = c.getLatestProposer()
				if err != nil {
					return nil, fmt.Errorf("get latest proposer: %w", err)
				}
			} else {
				return nil, fmt.Errorf("get batch at height: %w", err)
			}
		} else {
			proposerAddr = res.Batch.Sequencer
		}
	}

	if proposerAddr == "" || proposerAddr == SENTINEL_PROPOSER {
		return nil, settlement.ErrProposerIsSentinel
	}

	// Find and return the matching sequencer
	for _, seq := range seqs {
		if seq.SettlementAddress == proposerAddr {
			return &seq, nil
		}
	}
	return nil, fmt.Errorf("proposer not found")
}

// GetSequencerByAddress returns a sequencer by its address.
func (c *Client) GetSequencerByAddress(address string) (types.Sequencer, error) {
	var res *sequencertypes.QueryGetSequencerResponse
	req := &sequencertypes.QueryGetSequencerRequest{
		SequencerAddress: address,
	}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.sequencerQueryClient.Sequencer(c.ctx, req)
		if err == nil {
			return nil
		}

		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return types.Sequencer{}, err
	}

	dymintPubKey := protoutils.GogoToCosmos(res.Sequencer.DymintPubKey)
	var pubKey cryptotypes.PubKey
	err = c.protoCodec.UnpackAny(dymintPubKey, &pubKey)
	if err != nil {
		return types.Sequencer{}, err
	}

	tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
	if err != nil {
		return types.Sequencer{}, err
	}

	return *types.NewSequencer(
		tmPubKey,
		res.Sequencer.Address,
		res.Sequencer.RewardAddr,
		res.Sequencer.WhitelistedRelayers,
	), nil
}

// GetAllSequencers returns all sequencers of the given rollapp.
func (c *Client) GetAllSequencers() ([]types.Sequencer, error) {
	req := &sequencertypes.QueryGetSequencersByRollappRequest{
		RollappId:  c.rollappId,
		Pagination: &querytypes.PageRequest{},
	}

	res := []sequencertypes.Sequencer{}

	err := c.RunWithRetry(func() error {
		for {
			qres, err := c.sequencerQueryClient.SequencersByRollapp(c.ctx, req)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
				}
				return err
			}
			res = append(res, qres.Sequencers...)
			req.Pagination.Key = qres.GetPagination().NextKey
			if len(req.Pagination.Key) == 0 {
				return nil
			}
		}
	})
	if err != nil {
		return nil, err
	}
	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	var sequencerList []types.Sequencer
	for _, sequencer := range res {
		dymintPubKey := protoutils.GogoToCosmos(sequencer.DymintPubKey)
		var pubKey cryptotypes.PubKey
		err := c.protoCodec.UnpackAny(dymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}

		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
		if err != nil {
			return nil, err
		}

		sequencerList = append(sequencerList, *types.NewSequencer(
			tmPubKey,
			sequencer.Address,
			sequencer.RewardAddr,
			sequencer.WhitelistedRelayers,
		))
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
		dymintPubKey := protoutils.GogoToCosmos(sequencer.DymintPubKey)
		var pubKey cryptotypes.PubKey
		err := c.protoCodec.UnpackAny(dymintPubKey, &pubKey)
		if err != nil {
			return nil, err
		}

		tmPubKey, err := cryptocodec.ToTmPubKeyInterface(pubKey)
		if err != nil {
			return nil, err
		}
		sequencerList = append(sequencerList, *types.NewSequencer(
			tmPubKey,
			sequencer.Address,
			sequencer.RewardAddr,
			sequencer.WhitelistedRelayers,
		))
	}

	return sequencerList, nil
}

// GetNextProposer returns the next proposer on the hub.
// In case the current proposer is the next proposer, it returns nil.
// in case there is no next proposer, it returns an empty sequencer struct.
// in case there is a next proposer, it returns the next proposer.
func (c *Client) GetNextProposer() (*types.Sequencer, error) {
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
	if nextAddr == "" || nextAddr == SENTINEL_PROPOSER {
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

func (c *Client) GetRollapp() (*types.Rollapp, error) {
	var res *rollapptypes.QueryGetRollappResponse
	req := &rollapptypes.QueryGetRollappRequest{
		RollappId: c.rollappId,
	}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.cosmosClient.GetRollappClient().Rollapp(c.ctx, req)
		if err == nil {
			return nil
		}
		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get rollapp: %w", err)
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	rollapp := types.RollappFromProto(res.Rollapp)
	return &rollapp, nil
}

// GetObsoleteDrs returns the list of deprecated DRS.
func (c *Client) GetObsoleteDrs() ([]uint32, error) {
	var res *rollapptypes.QueryObsoleteDRSVersionsResponse
	req := &rollapptypes.QueryObsoleteDRSVersionsRequest{}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.cosmosClient.GetRollappClient().ObsoleteDRSVersions(c.ctx, req)
		if err == nil {
			return nil
		}
		if status.Code(err) == codes.NotFound {
			return retry.Unrecoverable(errors.Join(gerrc.ErrNotFound, err))
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get rollapp: %w", err)
	}

	// not supposed to happen, but just in case
	if res == nil {
		return nil, fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	return res.DrsVersions, nil
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
			Height:     block.Header.Height,
			StateRoot:  block.Header.AppHash[:],
			Timestamp:  block.Header.GetTimestamp(),
			DrsVersion: batch.DRSVersion[index],
		}
		blockDescriptors[index] = blockDescriptor
	}

	settlementBatch := &rollapptypes.MsgUpdateState{
		Creator:         addr,
		RollappId:       c.rollappId,
		StartHeight:     batch.StartHeight(),
		NumBlocks:       batch.NumBlocks(),
		DAPath:          daResult.SubmitMetaData.ToPath(),
		BDs:             rollapptypes.BlockDescriptors{BD: blockDescriptors},
		Last:            batch.LastBatch,
		RollappRevision: batch.Revision,
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

func (c *Client) getLatestProposer() (string, error) {
	var proposerAddr string
	err := c.RunWithRetry(func() error {
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
		return "", err
	}
	return proposerAddr, nil
}

// SubmitTEEAttestation submits a TEE attestation to fast-finalize state updates
func (c *Client) SubmitTEEAttestation(token string, nonce rollapptypes.TEENonce, finalizedIx, currIx uint64) error {
	account, err := c.cosmosClient.GetAccount(c.config.DymAccountName)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	addr, err := account.Address(addressPrefix)
	if err != nil {
		return fmt.Errorf("derive address: %w", err)
	}

	// Create the MsgFastFinalizeWithTEE message
	msg := &rollapptypes.MsgFastFinalizeWithTEE{
		Creator:          addr,
		AttestationToken: token,
		StateIndex:       currIx,
		Nonce:            nonce,
	}

	// Broadcast the transaction
	txResp, err := c.cosmosClient.BroadcastTx(c.config.DymAccountName, msg)
	if err != nil {
		return fmt.Errorf("broadcast TEE attestation tx: %w", err)
	}

	if txResp.Code != 0 {
		return fmt.Errorf("TEE attestation tx failed with code %d: %s", txResp.Code, txResp.RawLog)
	}

	return nil
}

func (c *Client) GetSignerBalance() (types.Balance, error) {
	account, err := c.cosmosClient.GetAccount(c.config.DymAccountName)
	if err != nil {
		return types.Balance{}, fmt.Errorf("obtain account: %w", err)
	}

	addr, err := account.Address(addressPrefix)
	if err != nil {
		return types.Balance{}, fmt.Errorf("derive address: %w", err)
	}

	denom := "adym"

	res, err := c.cosmosClient.GetBalance(c.ctx, addr, denom)
	if err != nil {
		return types.Balance{}, fmt.Errorf("get balance: %w", err)
	}

	balance := types.Balance{
		Amount: res.Amount,
		Denom:  res.Denom,
	}

	return balance, nil
}

func (c *Client) ValidateGenesisBridgeData(data rollapptypes.GenesisBridgeData) error {
	var res *rollapptypes.QueryValidateGenesisBridgeResponse
	req := &rollapptypes.QueryValidateGenesisBridgeRequest{
		RollappId: c.rollappId,
		Data:      data,
	}

	err := c.RunWithRetry(func() error {
		var err error
		res, err = c.cosmosClient.GetRollappClient().ValidateGenesisBridge(c.ctx, req)
		return err
	})
	if err != nil {
		return fmt.Errorf("rollapp client: validate genesis bridge: %w", err)
	}

	// not supposed to happen, but just in case
	if res == nil {
		return fmt.Errorf("empty response: %w", gerrc.ErrUnknown)
	}

	if !res.Valid || len(res.Err) != 0 {
		return fmt.Errorf("genesis bridge data is invalid: %s", res.Err)
	}

	return nil
}
