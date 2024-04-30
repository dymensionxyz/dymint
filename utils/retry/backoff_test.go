package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBackoff(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		b := NewBackoffConfig().Backoff()
		last := time.Duration(0)
		for range 9 { // growing
			d := b.Delay()
			require.Less(t, last, d)
			last = d
		}
		for range 4 { // maxed out
			d := b.Delay()
			require.Equal(t, last, d)
			last = d
		}
	})
	t.Run("decimal growth factor", func(t *testing.T) {
		initial := time.Second
		factor := 1.5
		b := NewBackoffConfig(WithInitialDelay(initial), WithGrowthFactor(factor), WithMaxDelay(0)).Backoff()
		b.Delay() // skip first so that last is initial
		last := initial
		for range 10 {
			d := b.Delay()
			require.Equal(t, time.Duration(float64(last)*factor), d)
			last = d
		}
	})
}
