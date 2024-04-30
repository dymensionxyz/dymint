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
	blockPauseTolerance time.Duration
	shouldProduceBlocks chan bool

	// cancel any scheduled pause that was set to happen in the future
	cancelFutureBlockPause func()
}

func makeNodeHealthErrorHandler(blockPauseTolerance time.Duration) nodeHealthErrorHandler {
	return nodeHealthErrorHandler{
		blockPauseTolerance:    blockPauseTolerance,
		shouldProduceBlocks:    make(chan bool, 1),
		cancelFutureBlockPause: func() {},
	}
}

// handle must not be called concurrently
func (h *nodeHealthErrorHandler) handle(err error) {
	h.cancelFutureBlockPause()

	if err == nil {
		// everything is fine!
		h.shouldProduceBlocks <- true // cancel any existing pause in production (NOTE: this line must go 2nd)
		return
	}

	h.cancelFutureBlockPause = utime.CancellableAfterFunc(h.blockPauseTolerance, func() {
		h.shouldProduceBlocks <- false
	})
}
