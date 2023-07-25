package celestia

const (
	perByteGasTolerance   = 2
	pfbGasFixedCost       = 80000
	defaultGasPerBlobByte = 8
)

func (c *DataAvailabilityLayerClient) calculateFees(gas uint64) int64 {
	fees := c.config.Fee
	if fees == 0 {
		fees = int64(c.config.GasPrices * float64(gas))
	}

	return fees
}

// EstimateGas estimates the gas required to pay for a set of blobs in a PFB.
func EstimateGas(blobSizes int) uint64 {
	totalByteCount := 0
	totalByteCount += blobSizes
	variableGasAmount := (defaultGasPerBlobByte + perByteGasTolerance) * totalByteCount
	return uint64(variableGasAmount + pfbGasFixedCost)
}
