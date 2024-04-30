package block

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFoo(t *testing.T) {
	/*
		It's not best practice to use such timers
		TODO: refactor
	*/
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
		time.Sleep(time.Millisecond * 200)
		require.Len(t, h.shouldProduceBlocksCh, 0) // must be cancelled
	})
	t.Run("unhealthy->healthy->unhealthy does not trigger too fast", func(t *testing.T) {
		/*
			Make sure that the healthy event COMPLETELY cancels out the first unhealthy event
		*/
		h := makeNodeHealthErrorHandler(time.Millisecond * 100)

		h.handle(errors.New("foo"))
		time.Sleep(time.Millisecond * 50)

		h.handle(nil) // healthy overrides unhealthy
		time.Sleep(time.Millisecond * 25)

		h.handle(errors.New("bar")) // another unhealthy one!

		ok := <-h.shouldProduceBlocksCh
		require.True(t, ok) // we got the healthy event

		time.Sleep(time.Millisecond * 80)

		require.Len(t, h.shouldProduceBlocksCh, 0) // first op must have been cancelled!

		time.Sleep(time.Millisecond * 200)
		ok = <-h.shouldProduceBlocksCh
		require.False(t, ok) // make sure we eventually get the second event
	})
}
