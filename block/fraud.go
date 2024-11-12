package block

import (
	"context"
)

// FraudHandler is an interface that defines a method to handle faults.
// Contract: should not be blocking.
type FraudHandler interface {
	// HandleFault handles a fault that occurred in the system.
	// The fault is passed as an error type.
	HandleFault(ctx context.Context, fault error)
}

// FreezeHandler is used to handle faults coming from executing and validating blocks.
// once a fault is detected, it publishes a DataHealthStatus event to the pubsub which sets the node in a frozen state.
type FreezeHandler struct {
	m *Manager
}

func (f FreezeHandler) HandleFault(ctx context.Context, fault error) {
	f.m.freezeNode(ctx, fault)
}

func NewFreezeHandler(manager *Manager) *FreezeHandler {
	return &FreezeHandler{
		m: manager,
	}
}
