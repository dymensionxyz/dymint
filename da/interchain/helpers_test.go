package interchain_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dymensionxyz/dymint/da/interchain"
	"github.com/dymensionxyz/dymint/types"
)

func FuzzEncodeDecodeBatch1(f *testing.F) {
	f.Add(uint64(0), uint64(0))

	f.Fuzz(func(t *testing.T, h1, h2 uint64) {
		// Generate batches with random headers
		expected := types.Batch{
			StartHeight: h1,
			EndHeight:   h2,
			Blocks:      []*types.Block{},
			Commits:     []*types.Commit{},
		}

		data, err := interchain.EncodeBatch(expected)
		require.NoError(t, err)

		actual, err := interchain.DecodeBatch(data)
		require.NoError(t, err)

		require.Equal(t, expected, actual)
	})
}

func TestEncodeDecodeBatch(t *testing.T) {
	expected := types.Batch{
		StartHeight: 1,
		EndHeight:   123,
		Blocks: []*types.Block{
			{
				Header: types.Header{
					Height: 1,
				},
			},
			{
				Header: types.Header{
					Height: 2,
				},
			},
		},
		Commits: []*types.Commit{
			{
				Height: 1,
			},
			{
				Height: 2,
			},
		},
	}

	data, err := interchain.EncodeBatch(expected)
	require.NoError(t, err)

	actual, err := interchain.DecodeBatch(data)
	require.NoError(t, err)

	require.True(t, reflect.DeepEqual(expected, actual))
	require.Equal(t, expected, actual)
}
