package settlement

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	tmpubsub "github.com/tendermint/tendermint/libs/pubsub"
	tmquery "github.com/tendermint/tendermint/libs/pubsub/query"
)

// Type keys

const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "settlement.event"
)

// Types

const (
	// EventNewBatchAccepted should be emitted internally in order to communicate between the settlement layer and the hub client
	EventNewBatchAccepted      = "EventNewBatchAccepted"
	EventSequencersListUpdated = "SequencersListUpdated"
	EventHealthStatus          = "SettlementHealthStatus"
)

// Convenience objects

var (
	HealthStatus     = map[string][]string{EventTypeKey: {EventHealthStatus}}     // TODO: rename with event somewhere
	NewBatchAccepted = map[string][]string{EventTypeKey: {EventNewBatchAccepted}} // TODO: rename with event somewhere
)

type EventDataNewBatchAccepted struct {
	// EndHeight is the height of the last accepted batch
	EndHeight uint64
	// StateIndex is the rollapp-specific index the batch was saved in the SL
	StateIndex uint64
}

type EventDataSequencersListUpdated struct {
	// Sequencers is the list of new sequencers
	Sequencers []types.Sequencer
}

type EventDataHealth struct {
	// Error is the error that was encountered in case of a health check failure, nil implies healthy
	Error error
}

// Queries
var (
	EventQueryNewSettlementBatchAccepted = QueryForEvent(EventNewBatchAccepted)
	EventQuerySettlementHealthStatus     = QueryForEvent(EventHealthStatus)
)

// QueryForEvent returns a query for the given event.
func QueryForEvent(eventType string) tmpubsub.Query {
	return tmquery.MustParse(fmt.Sprintf("%s='%s'", EventTypeKey, eventType))
}
