package da

import (
	"github.com/dymensionxyz/dymint/utils/event"
)

// Type keys
const (
	// EventTypeKey is a reserved composite key for event name.
	EventTypeKey = "da.event"
)

// Types

const (
	EventHealthStatus = "DAHealthStatus"
)

// Convenience objects

var EventHealthStatusList = map[string][]string{EventTypeKey: {EventHealthStatus}}

// Data

type EventDataHealth struct {
	// Error is the error that was encountered in case of a health check failure, nil implies healthy
	Error error
}

// Queries

var EventQueryDAHealthStatus = event.QueryFor(EventTypeKey, EventHealthStatus)
