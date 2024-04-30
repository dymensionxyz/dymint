package time

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCancellableAfterFunc(t *testing.T) {
	d := 10 * time.Millisecond
	t.Run("simple - func happens after some time", func(t *testing.T) {
		ok := atomic.Bool{}
		CancellableAfterFunc(d, func() {
			ok.Store(true)
		})
		// no cancel
		time.Sleep(2 * d)
		require.True(t, ok.Load())
	})
	t.Run("simple - can cancel", func(t *testing.T) {
		ok := atomic.Bool{}
		cancel := CancellableAfterFunc(d, func() {
			ok.Store(true)
		})
		cancel() // cancel well before the func is called
		time.Sleep(2 * d)
		require.False(t, ok.Load())
	})
	t.Run("race - if func started then cancel waits until its done", func(t *testing.T) {
		/*
			Note: should run this with race flag
		*/
		for range 100000 {
			go func() {
				cnt := atomic.Int64{}
				cancel := CancellableAfterFunc(d, func() {
					cnt.Add(1)
				})
				time.Sleep(d)
				cancel() // cancel around the same time as the func is running
				if cnt.Load() == 0 {
					cnt.Add(1)
				}
				require.Equal(t, int64(1), cnt.Load()) // Must never be 2!!
			}()
		}
	})
}
