package settlement

import (
	"github.com/dymensionxyz/dymint/types"
	uevent "github.com/dymensionxyz/dymint/utils/event"
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
	EventHealthStatusList     = map[string][]string{EventTypeKey: {EventHealthStatus}}
	EventNewBatchAcceptedList = map[string][]string{EventTypeKey: {EventNewBatchAccepted}}
)

// Data

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
	EventQueryNewSettlementBatchAccepted = uevent.QueryFor(EventTypeKey, EventNewBatchAccepted)
	EventQuerySettlementHealthStatus     = uevent.QueryFor(EventTypeKey, EventHealthStatus)
)
