package settlement

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Define the event type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "settlement.event"
)

// Define the event types
const (
	// EventNewSettlementBatchAccepted should be emitted internally in order to communicate between the settlement layer and the hub client
	EventNewSettlementBatchAccepted = "EventNewSettlementBatchAccepted"
	EventSequencersListUpdated      = "SequencersListUpdated"
	EventSettlementHealthStatus     = "SettlementHealthStatus"
)

// HealthEvent is a convenience object
var HealthEvent = map[string][]string{EventTypeKey: {EventSettlementHealthStatus}}

type EventDataNewBatchAccepted struct {
	// EndHeight is the height of the last accepted batch
	EndHeight uint64
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

type EventDataNewSettlementBatchAccepted EventDataNewBatchAccepted

type EventDataSequencersListUpdated struct {
	// Sequencers is the list of sequencers
	Sequencers []types.Sequencer
}

type EventDataHealth struct {
	// Error is the error that was encountered in case of a health check failure, nil implies healthy
	Error error
}

// Define queries
var (
	EventQueryNewSettlementBatchAccepted = QueryForEvent(EventNewSettlementBatchAccepted)
	EventQuerySettlementHealthStatus     = QueryForEvent(EventSettlementHealthStatus)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
