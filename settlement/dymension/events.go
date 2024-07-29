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
	eventStateUpdate          = "state_update.rollapp_id='%s' AND state_update.status='PENDING'"
	eventSequencersListUpdate = "create_sequencer.rollapp_id='%s'"
	eventRotationStarted      = "rotation_started.rollapp_id='%s'"
)

func (c *Client) getEventData(eventType string, rawEventData ctypes.ResultEvent) (interface{}, error) {
	switch eventType {
	case settlement.EventNewBatchAccepted:
		return convertToNewBatchEvent(rawEventData)
	}
	return nil, fmt.Errorf("event type %s not recognized", eventType)
}

func (c *Client) eventHandler() {
	subscriber := fmt.Sprintf("dymension-client-%s", uuid.New().String())

	// TODO: add validation callback for the event data
	eventMap := map[string]string{
		fmt.Sprintf(eventStateUpdate, c.config.RollappID): settlement.EventNewBatchAccepted,
	}

	combinedEventQuery := fmt.Sprintf(eventStateUpdate, c.config.RollappID)
	eventsChannel, err := c.cosmosClient.SubscribeToEvents(c.ctx, subscriber, combinedEventQuery, 1000)
	if err != nil {
		panic(err)
	}
	defer c.cosmosClient.UnsubscribeAll(c.ctx, subscriber)

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-c.cosmosClient.EventListenerQuit():
			// TODO(omritoptix): Fallback to polling
			panic("Settlement WS disconnected")
		case event := <-eventsChannel:
			// Assert value is in map and publish it to the event bus
			data, ok := eventMap[event.Query]
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
