package celestia

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	uretry "github.com/dymensionxyz/dymint/utils/retry"

	"github.com/stretchr/testify/assert"
)

func TestCreateConfig(t *testing.T) {
	mustMarshal := func(v any) []byte {
		bz, _ := json.Marshal(v)
		return bz
	}

	t.Run("simple", func(t *testing.T) {
		c := Config{
			BaseURL:       TestConfig.BaseURL,
			AppNodeURL:    TestConfig.AppNodeURL,
			Timeout:       TestConfig.Timeout,
			Fee:           0,
			GasPrices:     42,
			GasAdjustment: 42,
			GasLimit:      42,
			Backoff:       uretry.NewBackoffConfig(uretry.WithGrowthFactor(1.65)),
			RetryAttempts: 10,
			RetryDelay:    10 * time.Second,
		}
		bz := mustMarshal(c)
		gotC, err := createConfig(bz)
		require.NoError(t, err)
		assert.Equal(t, c.Backoff, gotC.Backoff)
		assert.Equal(t, c.RetryAttempts, gotC.RetryAttempts)
		assert.Equal(t, c.RetryDelay, gotC.RetryDelay)
	})
	t.Run("no backoff", func(t *testing.T) {
		c := Config{
			BaseURL:       TestConfig.BaseURL,
			AppNodeURL:    TestConfig.AppNodeURL,
			Timeout:       TestConfig.Timeout,
			Fee:           0,
			GasPrices:     42,
			GasAdjustment: 42,
			GasLimit:      42,
		}
		bz := mustMarshal(c)
		gotC, err := createConfig(bz)
		require.NoError(t, err)
		assert.Equal(t, defaultSubmitBackoff, gotC.Backoff)
	})
}
