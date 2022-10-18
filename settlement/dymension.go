package settlement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	
	"github.com/dymensionxyz/cosmosclient/cosmosclient"
	rollapptypes "github.com/dymensionxyz/dymension/x/rollapp/types"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/log"
	"github.com/dymensionxyz/dymint/types"
	"github.com/hashicorp/go-multierror"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/tendermint/tendermint/libs/pubsub"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	defaultBatchSize   = 5
	addressPrefix      = "dym"
	dymRollappVersion  = 0
	defaultNodeAddress = "http://localhost:26657"
)

const (
	eventStateUpdate = "state_update.rollapp_id='%s'"
)

// DymensionLayerClient is intended only for usage in tests.
type DymensionLayerClient struct {
	logger             log.Logger
	pubsub             *pubsub.Server
	latestHeight       uint64
	config             Config
	ctx                context.Context
	cancel             context.CancelFunc
	client             *cosmosclient.Client
	rollappQueryClient rollapptypes.QueryClient
	// eventsChannel is the channel that receives events from the dymension layer.
	eventsChannel <-chan ctypes.ResultEvent
	eventMap      map[string]string
}

// Config for the DymensionLayerClient
type Config struct {
	BatchSize      uint64                       `json:"batch_size"`
	KeyringBackend cosmosaccount.KeyringBackend `json:"keyring_backend"`
	NodeAddress    string                       `json:"node_address"`
	KeyRingHomeDir string                       `json:"keyring_home_dir"`
	DymAccountName string                       `json:"dym_account_name"`
	RollappID      string                       `json:"rollapp_id"`
}

var _ LayerClient = &DymensionLayerClient{}

// Init is called once. it initializes the struct members.
func (d *DymensionLayerClient) Init(config []byte, pubsub *pubsub.Server, logger log.Logger) error {
	c, err := d.getConfig(config)
	if err != nil {
		return err
	}
	d.config = *c
	d.pubsub = pubsub
	d.logger = logger
	d.ctx, d.cancel = context.WithCancel(context.Background())
	client, err := cosmosclient.New(
		d.ctx,
		cosmosclient.WithAddressPrefix(addressPrefix),
		cosmosclient.WithNodeAddress(c.NodeAddress),
		cosmosclient.WithKeyringBackend(c.KeyringBackend),
		cosmosclient.WithHome(c.KeyRingHomeDir),
	)
	if err != nil {
		return err
	}
	d.client = &client
	d.rollappQueryClient = rollapptypes.NewQueryClient(d.client.Context())
	// TODO(omritoptix): Build this map dynamically from our events.
	d.eventMap = map[string]string{
		fmt.Sprintf(eventStateUpdate, d.config.RollappID): EventNewSettlementBatchAccepted,
	}
	return nil
}

func (d *DymensionLayerClient) decodeConfig(config []byte) (*Config, error) {
	var c Config
	err := json.Unmarshal(config, &c)
	return &c, err
}

func (d *DymensionLayerClient) getConfig(config []byte) (*Config, error) {
	var c *Config
	if len(config) > 0 {
		var err error
		c, err = d.decodeConfig(config)
		if err != nil {
			return nil, err
		}
		if c.BatchSize == 0 {
			c.BatchSize = defaultBatchSize
		}
	} else {
		c = &Config{
			BatchSize:      defaultBatchSize,
			KeyringBackend: cosmosaccount.KeyringTest,
			NodeAddress:    defaultNodeAddress,
		}
	}
	return c, nil
}

// Start is called once, after init. It initializes the query client.
func (d *DymensionLayerClient) Start() error {
	d.logger.Debug("settlement Layer Client starting.")
	latestBatch, err := d.RetrieveBatch()
	if err != nil {
		if err == ErrBatchNotFound {
			return nil
		}
		return err
	}
	d.latestHeight = latestBatch.EndHeight
	d.logger.Info("Updated latest height from settlement layer", "latestHeight", d.latestHeight)

	// Subscribe to relevant events
	err = d.client.RPC.WSEvents.Start()
	if err != nil {
		return err
	}
	// TODO(omritoptix): eventsChannel should be a generic channel which is later filtered by the event type.
	d.eventsChannel, err = d.client.RPC.WSEvents.Subscribe(context.Background(), "dymension-client", fmt.Sprintf(eventStateUpdate, d.config.RollappID))
	if err != nil {
		return err
	}
	go d.eventHandler()

	return nil
}

// Stop is called once, after Start.
func (d *DymensionLayerClient) Stop() error {
	d.logger.Info("Settlement Layer Client stopping")
	err := d.client.RPC.WSEvents.Stop()
	if err != nil {
		return err
	}
	d.cancel()
	return nil
}

func (d *DymensionLayerClient) eventHandler() {
	for {
		select {
		case <-d.ctx.Done():
			return
		case event := <-d.eventsChannel:
			// Assert value is in map and publish it to the event bus
			d.logger.Debug("Received event from settlement layer", "event", event)
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
			err = d.pubsub.PublishWithEvents(d.ctx, eventData, map[string][]string{EventTypeKey: {d.eventMap[event.Query]}})
			if err != nil {
				d.logger.Error("Error publishing event", "err", err)
			}
		}
	}
}

func (d *DymensionLayerClient) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case EventNewSettlementBatchAccepted:
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
		NewBatchEvent := &EventDataNewSettlementBatchAccepted{
			EndHeight:  endHeight,
			StateIndex: uint64(stateIndex),
		}
		return NewBatchEvent, nil
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

func (d *DymensionLayerClient) validateBatch(batch *types.Batch) error {
	if batch.StartHeight != d.latestHeight+1 {
		return errors.New("batch start height must be last height + 1")
	}
	if batch.EndHeight < batch.StartHeight {
		return errors.New("batch end height must be greater or equal to start height")
	}
	return nil
}

func (d *DymensionLayerClient) convertBatchtoSettlementBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) (*rollapptypes.MsgUpdateState, error) {
	account, err := d.client.Account(d.config.DymAccountName)
	if err != nil {
		return nil, err
	}
	addr := account.Address(addressPrefix)
	DAMetaData := &DAMetaData{
		Height: daResult.DAHeight,
		// TODO(omritoptix): Change da to be a param
		Client: da.Mock,
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
		DAPath:      DAMetaData.toPath(),
		Version:     dymRollappVersion,
		BDs:         rollapptypes.BlockDescriptors{BD: blockDescriptors},
	}
	return settlementBatch, nil
}

// SubmitBatch submits the batch to the settlement layer. This should create a transaction which (potentially)
// triggers a state transition in the settlement layer.
func (d *DymensionLayerClient) SubmitBatch(batch *types.Batch, daResult *da.ResultSubmitBatch) *ResultSubmitBatch {
	d.logger.Debug("Submitting batch to settlement layer", "start height", batch.StartHeight, "end height", batch.EndHeight)
	// validate batch
	err := d.validateBatch(batch)
	if err != nil {
		return &ResultSubmitBatch{
			BaseResult: BaseResult{Code: StatusError, Message: err.Error()},
		}
	}
	// Build the result to save in the settlement layer.
	settlementBatch, err := d.convertBatchtoSettlementBatch(batch, daResult)
	if err != nil {
		return &ResultSubmitBatch{
			BaseResult: BaseResult{Code: StatusError, Message: err.Error()},
		}
	}
	// Send the batch to the settlement layer. stateIndex will be updated by an event.
	txResp, err := d.client.BroadcastTx(d.config.DymAccountName, settlementBatch)
	if err != nil || txResp.Code != 0 {
		d.logger.Error("Error sending batch to settlement layer", "error", err)
		return &ResultSubmitBatch{
			BaseResult: BaseResult{Code: StatusError, Message: err.Error()},
		}
	}
	d.logger.Info("Successfully submitted batch to settlement layer", "tx hash", txResp.TxHash)
	d.latestHeight = batch.EndHeight
	return &ResultSubmitBatch{
		BaseResult: BaseResult{Code: StatusSuccess},
	}
}

// RetrieveBatch Gets the batch which contains the given slHeight . Empty slHeight  returns the latest batch.
func (d *DymensionLayerClient) RetrieveBatch(stateIndex ...uint64) (*ResultRetrieveBatch, error) {
	var stateInfo rollapptypes.StateInfo
	if len(stateIndex) == 0 {
		d.logger.Debug("Getting latest batch from settlement layer", "latest height", d.latestHeight)
		latestStateInfoIndexResp, err := d.rollappQueryClient.LatestStateInfoIndex(d.ctx,
			&rollapptypes.QueryGetLatestStateInfoIndexRequest{RollappId: d.config.RollappID})
		if latestStateInfoIndexResp == nil {
			return nil, ErrBatchNotFound
		}
		if err != nil {
			return nil, err
		}
		stateInfoResp, err := d.rollappQueryClient.StateInfo(d.ctx,
			&rollapptypes.QueryGetStateInfoRequest{
				RollappId: d.config.RollappID, Index: latestStateInfoIndexResp.LatestStateInfoIndex.Index},
		)
		if err != nil {
			return nil, err
		}
		stateInfo = stateInfoResp.StateInfo
	} else if len(stateIndex) == 1 {
		d.logger.Debug("Getting batch from settlement layer", "state index", stateIndex)
		queryResp, err := d.rollappQueryClient.StateInfo(context.Background(), &rollapptypes.QueryGetStateInfoRequest{RollappId: d.config.RollappID, Index: stateIndex[0]})
		if err != nil {
			return nil, err
		}
		if queryResp == nil {
			return nil, ErrBatchNotFound
		}
		stateInfo = queryResp.StateInfo
	}
	daMetaData := &DAMetaData{}
	daMetaData, err := daMetaData.fromPath(stateInfo.DAPath)
	if err != nil {
		return nil, err
	}
	batchResult := &Batch{
		StartHeight: stateInfo.StartHeight,
		EndHeight:   stateInfo.StartHeight + stateInfo.NumBlocks - 1,
		MetaData: &BatchMetaData{
			DA: daMetaData,
		},
	}
	return &ResultRetrieveBatch{
		BaseResult: BaseResult{Code: StatusSuccess, StateIndex: stateInfo.StateInfoIndex.Index},
		Batch:      batchResult}, nil
}
