package events

import (
	"fmt"

	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Type Keys
const (
	// NodeTypeKey is a reserved composite key for event name.
	NodeTypeKey = "node.event"
)

//  Types

const (
	HealthStatus = "HealthStatus"
)

type DataHealthStatus struct {
	// Error is the error that was encountered in case of a health check failure. Nil implies both
	Error error
}

//  Queries

var QueryHealthStatus = QueryFor(HealthStatus)

// QueryFor returns a query for the given event.
func QueryFor(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", NodeTypeKey, eventType))
}
