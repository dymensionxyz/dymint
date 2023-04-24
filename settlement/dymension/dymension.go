package dymension

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sequencertypes "github.com/dymensionxyz/dymension/x/sequencer/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/hashicorp/go-multierror"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/pubsub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	addressPrefix      = "dym"
	dymRollappVersion  = 0
	defaultNodeAddress = "http://localhost:26657"
	defaultGasLimit    = 300000
)

const (
	eventStateUpdate          = "state_update.rollapp_id='%s'"
	eventSequencersListUpdate = "sequencers_list_update.rollapp_id='%s'"
)

// LayerClient is intended only for usage in tests.
type LayerClient struct {
	*settlement.BaseLayerClient
}

// Config for the DymensionLayerClient
type Config struct {
	KeyringBackend cosmosaccount.KeyringBackend `json:"keyring_backend"`
	NodeAddress    string                       `json:"node_address"`
	KeyRingHomeDir string                       `json:"keyring_home_dir"`
	DymAccountName string                       `json:"dym_account_name"`
	RollappID      string                       `json:"rollapp_id"`
	GasLimit       uint64                       `json:"gas_limit"`
	GasPrices      string                       `json:"gas_prices"`
	GasFees        string                       `json:"gas_fees"`
}

var _ settlement.LayerClient = &LayerClient{}

// Init is called once. it initializes the struct members.
func (dlc *LayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger, options ...settlement.Option) error {
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

// PostBatchResp is the response after posting a batch to the Dymension Hub.
type PostBatchResp struct {
	resp cosmosclient.Response
}

// GetTxHash returns the transaction hash.
func (d PostBatchResp) GetTxHash() string {
	return d.resp.TxHash
}

// GetCode returns the response code.
func (d PostBatchResp) GetCode() uint32 {
	return uint32(d.resp.Code)
}

// HubClient is the client for the Dymension Hub.
type HubClient struct {
	config               *Config
	logger               log.Logger
	pubsub               *pubsub.Server
	client               CosmosClient
	ctx                  context.Context
	cancel               context.CancelFunc
	rollappQueryClient   rollapptypes.QueryClient
	sequencerQueryClient sequencertypes.QueryClient
	protoCodec           *codec.ProtoCodec
	eventMap             map[string]string
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

func newDymensionHubClient(config []byte, pubsub *pubsub.Server, logger log.Logger, options ...Option) (*HubClient, error) {
	conf, err := getConfig(config)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, conf.RollappID):          settlement.EventNewSettlementBatchAccepted,
		fmt.Sprintf(eventSequencersListUpdate, conf.RollappID): settlement.EventSequencersListUpdated,
	}

	interfaceRegistry := cdctypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	protoCodec := codec.NewProtoCodec(interfaceRegistry)

	dymesionHubClient := &HubClient{
		config:     conf,
		logger:     logger,
		pubsub:     pubsub,
		ctx:        ctx,
		cancel:     cancel,
		eventMap:   eventMap,
		protoCodec: protoCodec,
	}

	for _, option := range options {
		option(dymesionHubClient)
	}

	if dymesionHubClient.client == nil {
		client, err := cosmosclient.New(
			ctx,
			getCosmosClientOptions(conf)...,
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

func decodeConfig(config []byte) (*Config, error) {
	var c Config
	err := json.Unmarshal(config, &c)
	return &c, err
}

func getConfig(config []byte) (*Config, error) {
	var c *Config
	if len(config) > 0 {
		var err error
		c, err = decodeConfig(config)
		if err != nil {
			return nil, err
		}
	} else {
		c = &Config{
			KeyringBackend: cosmosaccount.KeyringTest,
			NodeAddress:    defaultNodeAddress,
		}
	}
	return c, nil
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
	err := d.client.StopEventListener()
	if err != nil {
		return err
	}
	d.cancel()
	return nil
}

// PostBatch posts a batch to the Dymension Hub.
func (d *HubClient) PostBatch(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) (settlement.PostBatchResp, error) {
	msgUpdateState, err := d.convertBatchToMsgUpdateState(batch, daClient, daResult)
	if err != nil {
		return nil, err
	}
	txResp, err := d.client.BroadcastTx(d.config.DymAccountName, msgUpdateState)
	if err != nil || txResp.Code != 0 {
		d.logger.Error("Error sending batch to settlement layer", "error", err)
		return PostBatchResp{resp: txResp}, err
	}
	return PostBatchResp{resp: txResp}, nil
}

// GetLatestBatch returns the latest batch from the Dymension Hub.
func (d *HubClient) GetLatestBatch(rollappID string) (*settlement.ResultRetrieveBatch, error) {
	latestStateInfoIndexResp, err := d.rollappQueryClient.LatestStateInfoIndex(d.ctx,
		&rollapptypes.QueryGetLatestStateInfoIndexRequest{RollappId: d.config.RollappID})
	if latestStateInfoIndexResp == nil {
		return nil, settlement.ErrBatchNotFound
	}
	if err != nil {
		return nil, err
	}
	latestBatch, err := d.GetBatchAtIndex(rollappID, latestStateInfoIndexResp.LatestStateInfoIndex.Index)
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
		return nil, settlement.ErrNoSequencerForRollapp
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

func (d *HubClient) eventHandler() {
	// TODO(omritoptix): eventsChannel should be a generic channel which is later filtered by the event type.
	eventsChannel, err := d.client.SubscribeToEvents(context.Background(), "dymension-client", fmt.Sprintf(eventStateUpdate, d.config.RollappID))
	if err != nil {
		panic("Error subscribing to events")
	}
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-d.client.EventListenerQuit():
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
				d.logger.Error("Failed to get event data", "err", err)
				continue
			}
			err = d.pubsub.PublishWithEvents(d.ctx, eventData, map[string][]string{settlement.EventTypeKey: {d.eventMap[event.Query]}})
			if err != nil {
				d.logger.Error("Error publishing event", "err", err)
			}
		}
	}
}

func (d *HubClient) convertBatchToMsgUpdateState(batch *types.Batch, daClient da.Client, daResult *da.ResultSubmitBatch) (*rollapptypes.MsgUpdateState, error) {
	account, err := d.client.GetAccount(d.config.DymAccountName)
	if err != nil {
		return nil, err
	}
	addr := account.Address(addressPrefix)
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

func getCosmosClientOptions(config *Config) []cosmosclient.Option {
	if config.GasLimit == 0 {
		config.GasLimit = defaultGasLimit
	}
	options := []cosmosclient.Option{
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithNodeAddress(config.NodeAddress),
		cosmosclient.WithGasFees(config.GasFees),
		cosmosclient.WithGasLimit(config.GasLimit),
		cosmosclient.WithGasPrices(config.GasPrices),
	}
	if config.KeyringBackend != "" {
		options = append(options,
			cosmosclient.WithKeyringBackend(config.KeyringBackend),
			cosmosclient.WithHome(config.KeyRingHomeDir))
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
