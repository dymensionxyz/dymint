package events

import (
	"fmt"

	uevent "github.com/dymensionxyz/dymint/utils/event"
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

func (dhs DataHealthStatus) String() string {
	return fmt.Sprintf("DataHealthStatus{Error: %v}", dhs.Error)
}

//  Queries

var QueryHealthStatus = uevent.QueryFor(NodeTypeKey, HealthStatus)
