package events

import (
	"fmt"

	uevent "github.com/dymensionxyz/dymint/utils/event"
)

const (
	NodeTypeKey = "node.event"
)

const (
	HealthStatus = "HealthStatus"
)

var HealthStatusList = map[string][]string{NodeTypeKey: {HealthStatus}}

type DataHealthStatus struct {
	Error error
}

func (dhs DataHealthStatus) String() string {
	return fmt.Sprintf("DataHealthStatus{Error: %v}", dhs.Error)
}

var QueryHealthStatus = uevent.QueryFor(NodeTypeKey, HealthStatus)
