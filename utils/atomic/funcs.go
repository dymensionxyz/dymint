package atomic

import (
	"sync/atomic"
)

/*
TODO: move to sdk-utils
*/

// Uint64Sub does x := x-y and returns the new value of x
func Uint64Sub(x *atomic.Uint64, y uint64) uint64 {
	// Uses math
	return x.Add(^(y - 1))
}
