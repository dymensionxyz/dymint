package da

import "github.com/dymensionxyz/dymint/utilevent"

// Type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "da.event"
)

// Types

const (
	EventDAHealthStatus = "DAHealthStatus"
)

// Convenience objects

var HealthStatus = map[string][]string{EventTypeKey: {EventDAHealthStatus}} // TODO: rename

// Data

type EventDataHealth struct {
	// Error is the error that was encountered in case of a health check failure, nil implies healthy
	Error error
}

// Queries

var EventQueryDAHealthStatus = utilevent.QueryFor(EventTypeKey, EventDAHealthStatus)
