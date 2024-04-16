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

// Convenience objects

var HealthStatus = map[string][]string{EventTypeKey: {EventDAHealthStatus}} // TODO: rename

type EventDataHealth struct {
	// Error is the error that was encountered in case of a health check failure, nil implies healthy
	Error error
}

// Queries

var EventQueryDAHealthStatus = QueryForEvent(EventDAHealthStatus)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
