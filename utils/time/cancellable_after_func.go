package time

import "time"

// CancellableAfterFunc schedules a function to execute in future, and returns a function
// which can be used to cancel. It is possible that you may cancel after the function has
// already been executed. In that case, the cancel function will block until AFTER the function
// has finished.
func CancellableAfterFunc(d time.Duration, f func()) (blockingCancel func()) {
	c := make(chan struct{})
	t := time.AfterFunc(d, func() {
		f()
		close(c)
	})
	hasBeenCalled := false
	return func() {
		if hasBeenCalled {
			return
		}
		hasBeenCalled = true
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
