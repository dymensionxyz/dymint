package block

import (
	"context"

	"github.com/dymensionxyz/dymint/node/events"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// FraudHandler is an interface that defines a method to handle faults.
// Contract: should not be blocking.
type FraudHandler interface {
	// HandleFault handles a fault that occurred in the system.
	// The fault is passed as an error type.
	HandleFault(ctx context.Context, fault error)
}

// FreezeHandler is used to handle faults coming from executing and validating blocks.
type FreezeHandler struct {
	m *Manager
}

func (f FreezeHandler) HandleFault(ctx context.Context, fault error) {
	uevent.MustPublish(ctx, f.m.Pubsub, &events.DataHealthStatus{Error: fault}, events.HealthStatusList)
}

func NewFreezeHandler(manager *Manager) *FreezeHandler {
	return &FreezeHandler{
		m: manager,
	}
}
