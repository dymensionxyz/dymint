package types

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmptyNotEqualDefault(t *testing.T) {
	cons, _ := consMessagesHash(nil)
	extra := ExtraSignedData{
		ConsensusMessagesHash: cons,
	}
	h := extra.hash()
	require.False(t, bytes.Equal(h, defaultEvidenceHash))
}
