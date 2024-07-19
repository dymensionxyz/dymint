package channel

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
	C chan struct{} // Receive on C to sleep
}

func NewWaker() *Waker {
	return &Waker{make(chan struct{})}
}

// Wake up the waiting thread if any. Non blocking.
func (w Waker) Wake() {
	select {
	case w.C <- struct{}{}:
	default:
	}
}
