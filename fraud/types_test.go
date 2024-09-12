package fraud_test

import (
	"errors"
	"testing"

	"github.com/dymensionxyz/dymint/fraud"
)

type mockError struct {
	name string
	data string
}

func (m mockError) Error() string {
	return "some string"
}

func (mockError) Unwrap() error {
	return fraud.ErrFraud
}

func TestErrorIsErrFault(t *testing.T) {
	err := mockError{name: "test", data: "test"}

	if !errors.Is(err, fraud.ErrFraud) {
		t.Error("Expected Is to return true")
	}

	anotherErr := errors.New("some error")

	if errors.Is(anotherErr, fraud.ErrFraud) {
		t.Error("Expected Is to return false")
	}
}
