package dymension

import (
	"fmt"
	"strconv"

	"github.com/dymensionxyz/dymint/settlement"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/hashicorp/go-multierror"

	"github.com/google/uuid"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TODO: use types and attributes from dymension proto
const (
	eventStateUpdateFmt          = "state_update.rollapp_id='%s' AND state_update.status='PENDING'"
	eventSequencersListUpdateFmt = "create_sequencer.rollapp_id='%s'"
	eventRotationStartedFmt      = "proposer_rotation_started.rollapp_id='%s'"
)

func (c *Client) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case settlement.EventNewBatchAccepted:
		return convertToNewBatchEvent(rawEventData)
	case settlement.EventNewBondedSequencer:
		return convertToNewSequencerEvent(rawEventData)
	case settlement.EventRotationStarted:
		return convertToRotationStartedEvent(rawEventData)
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

func (c *Client) eventHandler() {
	subscriber := fmt.Sprintf("dymension-client-%s", uuid.New().String())

	eventStateUpdateQ := fmt.Sprintf(eventStateUpdateFmt, c.config.RollappID)
	eventSequencersListQ := fmt.Sprintf(eventSequencersListUpdateFmt, c.config.RollappID)
	eventRotationStartedQ := fmt.Sprintf(eventRotationStartedFmt, c.config.RollappID)

	// TODO: add validation callback for the event data
	eventMap := map[string]string{
		eventStateUpdateQ:     settlement.EventNewBatchAccepted,
		eventSequencersListQ:  settlement.EventNewBondedSequencer,
		eventRotationStartedQ: settlement.EventRotationStarted,
	}

	stateUpdatesC, err := c.cosmosClient.SubscribeToEvents(c.ctx, subscriber, eventStateUpdateQ, 1000)
	if err != nil {
		panic(fmt.Errorf("subscribe to events (%s): %w", eventStateUpdateQ, err))
	}
	sequencersListC, err := c.cosmosClient.SubscribeToEvents(c.ctx, subscriber, eventSequencersListQ, 1000)
	if err != nil {
		panic(fmt.Errorf("subscribe to events (%s): %w", eventSequencersListQ, err))
	}
	rotationStartedC, err := c.cosmosClient.SubscribeToEvents(c.ctx, subscriber, eventRotationStartedQ, 1000)
	if err != nil {
		panic(fmt.Errorf("subscribe to events (%s): %w", eventRotationStartedQ, err))
	}

	defer c.cosmosClient.UnsubscribeAll(c.ctx, subscriber)

	for {
		var e ctypes.ResultEvent
		select {
		case <-c.ctx.Done():
			return
		case <-c.cosmosClient.EventListenerQuit():
			// TODO(omritoptix): Fallback to polling
			return
		case e = <-stateUpdatesC:
		case e = <-sequencersListC:
		case e = <-rotationStartedC:
		}
		c.handleReceivedEvent(e, eventMap)
	}
}

func (c *Client) handleReceivedEvent(event ctypes.ResultEvent, eventMap map[string]string) {
	// Assert value is in map and publish it to the event bus
	internalType, ok := eventMap[event.Query]
	if !ok {
		c.logger.Error("Ignoring event. Type not supported", "event", event)
		return
	}
	eventData, err := c.getEventData(internalType, event)
	if err != nil {
		c.logger.Error("Error converting event data", "event", event, "error", err)
		return
	}

	c.logger.Debug("Publishing internal event", "event", internalType, "data", eventData)

	uevent.MustPublish(c.ctx, c.pubsub, eventData, map[string][]string{settlement.EventTypeKey: {internalType}})
}

func convertToNewBatchEvent(rawEventData ctypes.ResultEvent) (*settlement.EventDataNewBatchAccepted, error) {
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

func convertToNewSequencerEvent(rawEventData ctypes.ResultEvent) (*settlement.EventDataNewBondedSequencer, error) {
	// check all expected attributes  exists
	events := rawEventData.Events
	if events["create_sequencer.rollapp_id"] == nil {
		return nil, fmt.Errorf("missing expected attributes in event")
	}
	// TODO: validate rollappID

	if events["create_sequencer.sequencer"] == nil {
		return nil, fmt.Errorf("missing expected attributes in event")
	}

	return &settlement.EventDataNewBondedSequencer{
		SeqAddr: events["create_sequencer.sequencer"][0],
	}, nil
}

func convertToRotationStartedEvent(rawEventData ctypes.ResultEvent) (*settlement.EventDataRotationStarted, error) {
	// check all expected attributes  exists
	events := rawEventData.Events
	if events["proposer_rotation_started.rollapp_id"] == nil {
		return nil, fmt.Errorf("missing expected attributes in event")
	}

	// TODO: validate rollappID

	if events["proposer_rotation_started.next_proposer"] == nil {
		return nil, fmt.Errorf("missing expected attributes in event")
	}
	nextProposer := events["proposer_rotation_started.next_proposer"][0]
	rotationStartedEvent := &settlement.EventDataRotationStarted{
		NextSeqAddr: nextProposer,
	}
	return rotationStartedEvent, nil
}
