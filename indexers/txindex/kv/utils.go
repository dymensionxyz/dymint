package kv

import "github.com/google/orderedcode"

const TxEventHeightKey = "txevent.height"


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
