package block

import (
	"time"

	utime "github.com/dymensionxyz/dymint/utils/time"
)

type nodeHealthErrorHandler struct {
	// how long between a transition (node healthy -> unhealthy) before pausing block production.
	// if a node is unhealthy for longer than this, we stop producing blocks
	// note: in future, we could add the inverse for (node unhealthy -> healthy) to start producing
	//       blocks again, but for now we will do that instantly
	shouldProduceBlocksUnhealthyNodeTolerance time.Duration
	shouldProduceBlocksCh                     chan bool

	// cancel any scheduled pause that was set to happen in the future
	cancelScheduledBlockPause []func()
}

func makeNodeHealthErrorHandler(errorTolerance time.Duration) nodeHealthErrorHandler {
	return nodeHealthErrorHandler{
		shouldProduceBlocksUnhealthyNodeTolerance: errorTolerance,
		shouldProduceBlocksCh:                     make(chan bool, 1),
	}
}

// handle must not be called concurrently
func (h *nodeHealthErrorHandler) handle(err error) {
	if err == nil {
		// everything is fine!

		for _, cancel := range h.cancelScheduledBlockPause {
			cancel()
		}
		h.shouldProduceBlocksCh <- true // cancel any existing pause in production (NOTE: this line must go 2nd)
		return
	}

	h.cancelScheduledBlockPause = append(h.cancelScheduledBlockPause, utime.CancellableAfterFunc(h.shouldProduceBlocksUnhealthyNodeTolerance, func() {
		h.shouldProduceBlocksCh <- false
	}))
}
