package fraud

import (
	"context"
	"errors"
	"fmt"
)

// Handler is an interface that defines a method to handle faults.
// Contract: should not be blocking.
type Handler interface {
	// HandleFault handles a fault that occurred in the system.
	// The fault is passed as an error type.
	HandleFault(ctx context.Context, fault error)
}

type GenericHandler struct {
	handlers map[error]Handler
}

func NewGenericHandler() *GenericHandler {
	return &GenericHandler{
		handlers: make(map[error]Handler),
	}
}

func (gh *GenericHandler) RegisterHandler(errorType error, handler Handler) error {
	if errorType == nil {
		return errors.New("error type cannot be nil")
	}
	if handler == nil {
		return errors.New("handler cannot be nil")
	}
	if !errors.Is(errorType, ErrFraud) {
		return fmt.Errorf("error type must be or wrap ErrFraud")
	}
	if _, exists := gh.handlers[errorType]; exists {
		return fmt.Errorf("handler for error type %v already registered", errorType)
	}
	gh.handlers[errorType] = handler
	return nil
}

func (gh *GenericHandler) HandleFault(ctx context.Context, fault error) {
	for errType, handler := range gh.handlers {
		if errors.Is(fault, errType) {
			handler.HandleFault(ctx, fault)
			return
		}
	}
	panic(fmt.Sprintf("No handler registered for error type: %T", fault))
}
