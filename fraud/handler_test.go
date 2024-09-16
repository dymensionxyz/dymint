package fraud_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dymensionxyz/dymint/fraud"
)

type testError struct {
	msg string
}

func (e testError) Error() string {
	return e.msg
}

func (e testError) Unwrap() error {
	return fraud.ErrFraud
}

type testHandler struct {
	called     bool
	handledErr error
}

func (h *testHandler) HandleFault(ctx context.Context, fault error) {
	h.called = true
	h.handledErr = fault
}

func TestGenericHandler(t *testing.T) {
	t.Skip("Skipping test for now")

	t.Run("RegisterHandler", func(t *testing.T) {
		gh := fraud.NewGenericHandler()

		// Test registering a valid error
		validErr := testError{"valid error"}
		err := gh.RegisterHandler(validErr, &testHandler{})
		if err != nil {
			t.Errorf("Failed to register valid error: %v", err)
		}

		// Test registering an invalid error
		invalidErr := errors.New("invalid error")
		err = gh.RegisterHandler(invalidErr, &testHandler{})
		if err == nil {
			t.Error("Expected error when registering invalid error, got nil")
		}

		// Test registering a nil error
		err = gh.RegisterHandler(nil, &testHandler{})
		if err == nil {
			t.Error("Expected error when registering nil error, got nil")
		}

		// Test registering a nil handler
		err = gh.RegisterHandler(validErr, nil)
		if err == nil {
			t.Error("Expected error when registering nil handler, got nil")
		}
	})

	t.Run("HandleFault", func(t *testing.T) {
		gh := fraud.NewGenericHandler()

		err1 := testError{"error 1"}
		err2 := testError{"error 2"}

		handler1 := &testHandler{}

		_ = gh.RegisterHandler(testError{}, handler1)

		// Test handling a registered error
		gh.HandleFault(context.Background(), err1)
		if !handler1.called {
			t.Error("Handler1 was not called")
		}
		if handler1.handledErr != err1 {
			t.Errorf("Handler1 received wrong error. Got %v, want %v", handler1.handledErr, err1)
		}

		handler1.called = false

		gh.HandleFault(context.Background(), err2)
		if !handler1.called {
			t.Error("Handler1 was not called")
		}
		if handler1.handledErr != err2 {
			t.Errorf("Handler1 received wrong error. Got %v, want %v", handler1.handledErr, err2)
		}

		// Test handling an unregistered error
		defer func() {
			if r := recover(); r == nil {
				t.Error("The code did not panic for unregistered error type")
			}
		}()
		gh.HandleFault(context.Background(), errors.New("unregistered error"))
	})

	t.Run("Fails when registering same type two times even if different implementations", func(t *testing.T) {
		gh := fraud.NewGenericHandler()

		err1 := testError{"error 1"}
		err2 := testError{"error 2"}
		handler1 := &testHandler{}
		handler2 := &testHandler{}

		err := gh.RegisterHandler(err1, handler1)
		if err != nil {
			t.Errorf("Failed to register handler1: %v", err)
		}
		// Registering the same error type with a different handler should fail
		err = gh.RegisterHandler(err2, handler2)
		if err == nil {
			t.Errorf("Failed to register handler2: %v", err)
		}
	})

	t.Run("Handles wrapped errors", func(t *testing.T) {
		gh := fraud.NewGenericHandler()
		baseErr := testError{"base error"}
		wrappedErr := fmt.Errorf("wrapped: %w", baseErr)
		handler := &testHandler{}

		err := gh.RegisterHandler(testError{}, handler)
		if err != nil {
			t.Fatalf("Failed to register handler: %v", err)
		}

		gh.HandleFault(context.Background(), wrappedErr)
		if !handler.called {
			t.Error("Handler was not called for wrapped error")
		}
		if !errors.Is(handler.handledErr, baseErr) {
			t.Error("Handled error should unwrap to base error")
		}
	})
}
