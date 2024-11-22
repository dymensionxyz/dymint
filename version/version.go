package version

import (
	"fmt"
	"strconv"
)

var (
	Build = "<version>"
	DRS   = "0"
)

func GetDRSVersion() (uint32, error) {
	currentDRS, err := strconv.ParseUint(DRS, 10, 32)
	if err != nil {
		return uint32(0), fmt.Errorf("converting DRS version to int: %v", err)
	}
	return uint32(currentDRS), nil //nolint:gosec // DRS is uint32
}
