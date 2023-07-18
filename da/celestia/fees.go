package celestia

import sdktypes "github.com/cosmos/cosmos-sdk/types"

const (
	perByteGasTolerance   = 2
	pfbGasFixedCost       = 80000
	defaultGasPerBlobByte = 8
)

func (c *DataAvailabilityLayerClient) calculateFees(gas uint64) int64 {
	fees := c.config.Fee
	if fees == 0 {
		decGasPrice, _ := sdktypes.NewDecFromStr(c.config.GasPrices)
		adjustedPrice := decGasPrice.MustFloat64() * gasAdjustment
		fees = int64(adjustedPrice * float64(gas))
	}

	return fees
}

// EstimateGas estimates the gas required to pay for a set of blobs in a PFB.
func EstimateGas(blobSizes []int) uint64 {
	totalByteCount := 0
	for _, size := range blobSizes {
		totalByteCount += size
	}
	variableGasAmount := (defaultGasPerBlobByte + perByteGasTolerance) * totalByteCount

	return uint64(variableGasAmount + pfbGasFixedCost)
}
