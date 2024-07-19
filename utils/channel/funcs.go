package channel

import "context"

func DrainForever[T any](chs ...<-chan T) {
	for _, ch := range chs {
		go func() {
			for {
				<-ch
			}
		}()
	}
}

// Waker can be used to make a goroutine ('A') sleep, and have another goroutine ('B') wake him up
// A will not block if B is not asleep.
type Waker struct {
	C chan struct{}
}

func NewWaker() *Waker {
	return &Waker{make(chan struct{})}
}

// Wait until context is done, or awoken.
func (w Waker) Wait(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-w.C:
	default:
	}
}

// Wake up the waiting thread if any. Non blocking.
func (w Waker) Wake() {
	select {
	case w.C <- struct{}{}:
	default:
	}
}
