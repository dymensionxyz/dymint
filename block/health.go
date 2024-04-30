package block

import (
	"sync/atomic"
	"time"
)

type nodeHealthErrorHandler struct {
	// how long between a transition (node healthy -> unhealthy) before pausing block production.
	// if a node is unhealthy for longer than this, we stop producing blocks
	// note: in future, we could add the inverse for (node unhealthy -> healthy) to start producing
	//       blocks again, but for now we will do that instantly
	shouldProduceBlocksUnhealthyNodeTolerance time.Duration
	shouldProduceBlocksCh                     chan bool

	// Something we will send on the channel in future
	futureValueToSend atomic.Bool
}

func makeNodeHealthErrorHandler(errorTolerance time.Duration) nodeHealthErrorHandler {
	return nodeHealthErrorHandler{
		shouldProduceBlocksUnhealthyNodeTolerance: errorTolerance,
		shouldProduceBlocksCh:                     make(chan bool, 1),
	}
}

func (h *nodeHealthErrorHandler) handle(err error) {
	/*
		This implementation ended up being simpler than trying to use timer cancelling.
	*/

	if err == nil {
		// everything is fine!
		h.futureValueToSend.Store(true) // cancel any scheduled pause in production
		h.shouldProduceBlocksCh <- true // cancel any existing pause in production
		return
	}

	h.futureValueToSend.Store(false)
	time.AfterFunc(h.shouldProduceBlocksUnhealthyNodeTolerance, func() {
		h.shouldProduceBlocksCh <- h.futureValueToSend.Load()
	})
}
