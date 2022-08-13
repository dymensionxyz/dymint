package settlement

import (
	"fmt"

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
	EventNewSettlementBatchAccepted = "NewSettlementBatchAccepted"
)

// EventDataNewSettlementBatchAccepted defines the structure of the event data for the EventNewSettlementBatchAccepted
type EventDataNewSettlementBatchAccepted struct {
	// Batch is the settlement batch that was accepted
	Batch *Batch
}

// Define queries
var (
	EventQueryNewSettlementBatchAccepted = QueryForEvent(EventNewSettlementBatchAccepted)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
