package kv

import "github.com/google/orderedcode"

const TxEventHeightKey = "txevent.height"

// IntInSlice returns true if a is found in the list.
func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func eventHeightKey(height int64) ([]byte, error) {
	return orderedcode.Append(
		nil,
		TxEventHeightKey,
		height,
	)
}
