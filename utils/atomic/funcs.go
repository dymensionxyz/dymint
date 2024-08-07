package atomic

import (
	"sync/atomic"
)

func Uint64Sub(x *atomic.Uint64, y uint64) uint64 {
	return x.Add(^(y - 1))
}
