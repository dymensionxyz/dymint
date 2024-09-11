package fraud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dymensionxyz/dymint/fraud"
	"github.com/dymensionxyz/dymint/types"
)

func TestFault_Error(t *testing.T) {
	underlyingErr := errors.New("test error")
	fault := fraud.NewFault(underlyingErr, nil)

	expected := "fraud error: test error"
	if fault.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, fault.Error())
	}
}

func TestFault_Is(t *testing.T) {
	underlyingErr := errors.New("test error")
	fault1 := fraud.NewFault(underlyingErr, nil)
	fault2 := fraud.NewFault(underlyingErr, nil)
	otherErr := errors.New("other error")

	if !fault1.Is(fault2) {
		t.Error("Expected fault1.Is(fault2) to be true")
	}

	if fault1.Is(otherErr) {
		t.Error("Expected fault1.Is(otherErr) to be false")
	}
}

func TestFault_Unwrap(t *testing.T) {
	underlyingErr := errors.New("test error")
	fault := fraud.NewFault(underlyingErr, nil)

	if fault.Unwrap() != underlyingErr {
		t.Error("Expected Unwrap() to return the underlying error")
	}
}

func TestFault_GovProposalContent(t *testing.T) {
	underlyingErr := errors.New("test error")
	fault := fraud.NewFault(underlyingErr, nil)

	expected := "fraud error: test error"
	if fault.GovProposalContent() != expected {
		t.Errorf("Expected GovProposalContent '%s', got '%s'", expected, fault.GovProposalContent())
	}
}

func TestNewFault(t *testing.T) {
	underlyingErr := errors.New("test error")
	block := &types.Block{}
	fault := fraud.NewFault(underlyingErr, block)

	if fault.Underlying != underlyingErr {
		t.Error("Expected Underlying error to match")
	}

	if fault.Block != block {
		t.Error("Expected Block to match")
	}
}

func TestIsFault(t *testing.T) {
	underlyingErr := errors.New("test error")
	fault := fraud.NewFault(underlyingErr, nil)

	isFault := fraud.IsFault(fault)
	if !isFault {
		t.Error("Expected IsFault to return true for a Fault")
	}

	isFault = fraud.IsFault(errors.New("not a fault"))
	if isFault {
		t.Error("Expected IsFault to return false for a non-Fault error")
	}
}

type mockHandler struct {
	called bool
}

func (m *mockHandler) HandleFault(ctx context.Context, fault fraud.Fault) {
	m.called = true
}

func TestHandler(t *testing.T) {
	handler := &mockHandler{}
	ctx := context.Background()
	fault := fraud.NewFault(errors.New("test error"), nil)

	handler.HandleFault(ctx, fault)

	if !handler.called {
		t.Error("Expected HandleFault to be called")
	}
}
