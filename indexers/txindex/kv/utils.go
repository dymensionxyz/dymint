package kv

import "slices"

import "github.com/google/orderedcode"

const TxEventHeightKey = "txevent.height"

// IntInSlice returns true if a is found in the list.
func intInSlice(a int, list []int) bool {
	return slices.Contains(list, a)
}

func eventHeightKey(height int64) ([]byte, error) {
	return orderedcode.Append(
		nil,
		TxEventHeightKey,
		height,
	)
}
