package ioutils_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/da/interchain/ioutils"
)

func FuzzGzipGunzip(f *testing.F) {
	f.Add([]byte("YW5vdGhlciBlbmNvZGUgc3RyaW5n")) // base64 string
	f.Add([]byte("Different String!"))            // plain string
	f.Add([]byte("1234567890"))                   // numbers
	f.Add([]byte{})                               // empty slice

	f.Fuzz(func(t *testing.T, data []byte) {
		// Encode to gzip
		encoded, err := ioutils.Gzip(data)
		require.NoError(t, err)

		// Verify if it's a gzip
		ok := ioutils.IsGzip(encoded)
		require.True(t, ok)

		// Decode from gzip
		decoded, err := ioutils.Gunzip(encoded)
		require.NoError(t, err)

		// Check if the resulted output is not a gzip
		ok = ioutils.IsGzip(decoded)
		require.False(t, ok)

		// Compare the original data against the output
		require.Equal(t, data, decoded)
	})
}

func TestGzipGunzip(t *testing.T) {
	// Prepare the input
	var expected = []byte("Hello world!")

	// Encode to gzip
	encoded, err := ioutils.Gzip(expected)
	require.NoError(t, err)

	// Check the output is correct
	ok := ioutils.IsGzip(encoded)
	require.True(t, ok)

	// Decode from gzip
	decoded, err := ioutils.Gunzip(encoded)
	require.NoError(t, err)

	// The output is not gzip anymore
	ok = ioutils.IsGzip(decoded)
	require.False(t, ok)

	// Compare the input against the output
	require.Equal(t, expected, decoded)
}
