package time

import "time"

func CancellableAfterFunc(d time.Duration, f func()) (blockingCancel func()) {
	c := make(chan struct{})
	t := time.AfterFunc(d, func() {
		f()
		close(c)
	})
	return func() {
		/*
		 'For a timer created with AfterFunc(d, f), if t.Stop returns false, then the timer
		 has already expired and the function f has been started in its own goroutine;
		 Stop does not wait for f to complete before returning.
		 If the caller needs to know whether f is completed, it must coordinate
		 with f explicitly.'
		*/
		didStop := t.Stop()
		if !didStop {
			<-c // block until we're done
		}
	}
}
