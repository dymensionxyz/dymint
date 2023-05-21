package da

import (
	"fmt"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Define the event type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "da.event"
)

// Define the event types
const (
	EventDAHealthStatus = "DAHealthStatus"
)

// EventDataDAHealthStatus defines the structure of the event data for the EventDataDAHealthStatus
type EventDataDAHealthStatus struct {
	// Healthy is true if the da layer is healthy
	Healthy bool
	// Error is the error that was encountered in case of a health check failure
	Error error
}

// Define queries
var (
	EventQueryDAHealthStatus = QueryForEvent(EventDAHealthStatus)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
