package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchSerialization(t *testing.T) {
	batch := Batch{}
	bz, err := batch.MarshalBinary()
	require.Nil(t, err)
	require.Empty(t, bz)
}
