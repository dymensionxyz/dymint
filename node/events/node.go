package events

import (
	"fmt"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Define the event type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventNodeTypeKey = "node.event"
)

// Define the event types
const (
	EventHealthStatus = "HealthStatus"
)

// EventDataHealthStatus defines the structure of the event data for the EventHealthStatus
type EventDataHealthStatus struct {
	// Healthy is true if the base layers are healthy
	Healthy bool
	// Error is the error that was encountered in case of a health check failure
	Error error
}

// Define queries
var (
	EventQueryHealthStatus = QueryForEvent(EventHealthStatus)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventNodeTypeKey, eventType))
}
