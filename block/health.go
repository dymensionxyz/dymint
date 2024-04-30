package block

import (
	"time"
)

type nodeHealthErrorHandler struct {
	// how long between a transition (node healthy -> unhealthy) before pausing block production.
	// if a node is unhealthy for longer than this, we stop producing blocks
	// if it becomes healthy again, the timer is cancelled
	// note: in future, we could add the inverse for (node unhealthy -> healthy) to start producing
	//       blocks again, but for now we will do that instantly
	shouldProduceBlocksUnhealthyNodeTolerance time.Duration
	shouldProduceBlocksCh                     chan bool

	delayedFunc        time.Timer
	cancelDelayedPause func() bool
}

func (h *nodeHealthErrorHandler) handle(err error) {
	if err == nil { // everything is fine!
		if h.cancelDelayedPause != nil {
			// if we were unhealthy recently, make sure we dont stop producing blocks
			h.cancelDelayedPause()
		}
		h.shouldProduceBlocksCh <- true
		return
	}

	t := time.AfterFunc(h.shouldProduceBlocksUnhealthyNodeTolerance, func() {
		h.shouldProduceBlocksCh <- false
	})
	h.cancelDelayedPause = t.Stop
}
