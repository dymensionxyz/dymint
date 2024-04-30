package block

import "time"

type nodeHealthStatusHandler struct {
	// how long between a transition (node healthy -> unhealthy) before pausing block production.
	// if a node is unhealthy for longer than this, we stop producing blocks
	// if it becomes healthy again, the timer is cancelled
	// note: in future, we could add the inverse for (node unhealthy -> healthy) to start producing
	//       blocks again, but for now we will do that instantly
	shouldProduceBlocksUnhealthyNodeTolerance time.Duration
	shouldProduceBlocksCh                     chan bool
}
