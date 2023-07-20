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
	// This event should be emitted internally in order to communicate between the settlement layer and the hub client
	EventNewSettlementBatchAccepted = "EventNewSettlementBatchAccepted"
	// This event should be emitted externally when a batch is accepted
	EventNewBatchAccepted       = "EventNewBatchAccepted"
	EventSequencersListUpdated  = "SequencersListUpdated"
	EventSettlementHealthStatus = "SettlementHealthStatus"
)

// EventDataNewBatchAccepted defines the structure of the event data for the EventNewBatchAccepted
type EventDataNewBatchAccepted struct {
	// EndHeight is the height of the last accepted batch
	EndHeight uint64
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

// EventDataNewSettlementBatchAccepted defines the structure of the event data for the EventNewSettlementBatchAccepted
type EventDataNewSettlementBatchAccepted EventDataNewBatchAccepted

// EventDataSequencersListUpdated defines the structure of the event data for the EventSequencersListUpdated
type EventDataSequencersListUpdated struct {
	// Sequencers is the list of sequencers
	Sequencers []types.Sequencer
}

// EventDataSettlementHealthStatus defines the structure of the event data for the EventDataSettlementHealthStatus
type EventDataSettlementHealthStatus struct {
	// Healthy is true if the settlement layer is healthy
	Healthy bool
	// Error is the error that was encountered in case of a health check failure
	Error error
}

// Define queries
var (
	EventQueryNewSettlementBatchAccepted = QueryForEvent(EventNewSettlementBatchAccepted)
	EventQueryNewBatchAccepted           = QueryForEvent(EventNewBatchAccepted)
	EventQuerySettlementHealthStatus     = QueryForEvent(EventSettlementHealthStatus)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
