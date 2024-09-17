package fraud

import (
	"context"
)

// Handler is an interface that defines a method to handle faults.
// Contract: should not be blocking.
type Handler interface {
	// HandleFault handles a fault that occurred in the system.
	// The fault is passed as an error type.
	HandleFault(ctx context.Context, fault error)
}
