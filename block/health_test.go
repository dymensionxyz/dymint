package block

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		h := makeNodeHealthErrorHandler(time.Millisecond * 100)

		h.handle(nil)
		ok := <-h.shouldProduceBlocksCh
		require.True(t, ok)
	})
	t.Run("unhealthy", func(t *testing.T) {
		h := makeNodeHealthErrorHandler(time.Millisecond * 100)

		h.handle(errors.New("foo"))
		ok := <-h.shouldProduceBlocksCh
		require.False(t, ok)
	})
	t.Run("healthy overrides unhealthy", func(t *testing.T) {
		h := makeNodeHealthErrorHandler(time.Millisecond * 100)

		h.handle(errors.New("foo"))
		h.handle(nil) // healthy overrides unhealthy
		ok := <-h.shouldProduceBlocksCh
		require.True(t, ok)
	})
}
