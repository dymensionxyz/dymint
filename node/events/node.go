package events

import (
	"fmt"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// The event type keys
const (
	// EventNodeTypeKey is a reserved composite key for event name.
	EventNodeTypeKey = "node.event"
)

//  The event types

const (
	EventHealthStatus = "HealthStatus"
)

type EventDataHealthStatus struct {
	// Error is the error that was encountered in case of a health check failure. Nil implies both
	Error error
}

//  Queries

var EventQueryHealthStatus = QueryForEvent(EventHealthStatus)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventNodeTypeKey, eventType))
}
