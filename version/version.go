package version

import (
	"fmt"
	"strconv"
)

var (
	BuildVersion = "<version>"
	DrsVersion   = "0"
)

func CurrentDRSVersion() (uint32, error) {
	currentDRS, err := strconv.ParseUint(DrsVersion, 10, 32)
	if err != nil {
		return uint32(0), fmt.Errorf("converting DRS version to int: %v", err)
	}
	return uint32(currentDRS), nil
}
