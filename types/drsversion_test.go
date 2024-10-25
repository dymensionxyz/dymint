package types_test

import (
	"testing"

	"github.com/dymensionxyz/dymint/testutil"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/stretchr/testify/require"
)

func Test_DRS(t *testing.T) {

	drs := &types.DRSVersionHistory{}

	// set upgrade height
	upgrades := make(map[int]struct{})
	upgrades[5] = struct{}{}
	upgrades[13] = struct{}{}

	// create versions
	versions := make([]string, len(upgrades)+1)
	versionList := make([]string, 20)
	for i := 0; i < len(versions); i++ {
		version, err := testutil.CreateRandomVersionCommit()
		require.NoError(t, err)
		versions[i] = version
	}

	// populate version list per height
	j := 0
	for i := 0; i < 20; i++ {
		versionList[i] = versions[j]
		_, ok := upgrades[i]
		if ok {
			j++
		}
	}

	// add drs version to history and check whether its added
	for i := 0; i < 20; i++ {
		ok := drs.AddDRSVersion(uint64(i), versionList[i])
		if i > 0 && versionList[i] == versionList[i-1] {
			require.False(t, ok)
		} else {
			require.True(t, ok)
		}
	}

	// check getting drs version per height
	for height := uint64(0); height < uint64(20); height++ {
		version, err := drs.GetDRSVersion(uint64(height))
		require.NoError(t, err)
		require.Equal(t, versionList[height], version)
	}

	// test clearing versions
	drs.ClearDRSVersionHeights(6)

	// test successful after clearing
	version, err := drs.GetDRSVersion(uint64(10))
	require.NoError(t, err)
	require.Equal(t, versionList[10], version)

	// test not found after clearing
	_, err = drs.GetDRSVersion(uint64(5))
	require.ErrorIs(t, err, gerrc.ErrNotFound)

}
