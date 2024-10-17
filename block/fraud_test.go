package block_test

import (
	"errors"
	"testing"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

type mockError struct {
	name string
	data string
}

func (m mockError) Error() string {
	return "some string"
}

func (mockError) Unwrap() error {
	return gerrc.ErrFault
}

func TestErrorIsErrFault(t *testing.T) {
	err := mockError{name: "test", data: "test"}

	if !errors.Is(err, gerrc.ErrFault) {
		t.Error("Expected Is to return true")
	}

	anotherErr := errors.New("some error")

	if errors.Is(anotherErr, gerrc.ErrFault) {
		t.Error("Expected Is to return false")
	}
}
