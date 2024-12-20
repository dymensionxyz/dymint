package types

import (
	"testing"

	proto "github.com/gogo/protobuf/types"
)

func TestConsMessagesHashNilAndZeroLength(t *testing.T) {
	// Compute hash with nil
	hashNil := ConsMessagesHash(nil)

	// Compute hash with zero-length slice
	hashZeroLength := ConsMessagesHash([]*proto.Any{})

	// Compare the two hashes
	if hashNil != hashZeroLength {
		t.Errorf("Expected hashes to be the same, but got different hashes: %x and %x", hashNil, hashZeroLength)
	}
}
