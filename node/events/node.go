package events

import (
	"github.com/dymensionxyz/dymint/utilevent"
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

// Convenience

var HealthStatusList = map[string][]string{NodeTypeKey: {HealthStatus}}

type DataHealthStatus struct {
	// Error is the error that was encountered in case of a health check failure. Nil implies healthy.
	Error error
}

//  Queries

var QueryHealthStatus = utilevent.QueryFor(NodeTypeKey, HealthStatus)
