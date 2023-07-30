package celestia

import (
	"github.com/dymensionxyz/dymint/da/celestia/types"
)

const (
	ShareSize = types.ShareSize

	// PFBGasFixedCost is a rough estimate for the "fixed cost" in the gas cost
	// formula: gas cost = gas per byte * bytes per share * shares occupied by
	// blob + "fixed cost". In this context, "fixed cost" accounts for the gas
	// consumed by operations outside the blob's GasToConsume function (i.e.
	// signature verification, tx size, read access to accounts).
	//
	// Since the gas cost of these operations is not easy to calculate, linear
	// regression was performed on a set of observed data points to derive an
	// approximate formula for gas cost. Assuming gas per byte = 8 and bytes per
	// share = 512, we can solve for "fixed cost" and arrive at 65,000. gas cost
	// = 8 * 512 * number of shares occupied by the blob + 65,000 has a
	// correlation coefficient of 0.996. To be conservative, we round up "fixed
	// cost" to 75,000 because the first tx always takes up 10,000 more gas than
	// subsequent txs.
	PFBGasFixedCost = 75000

	// BytesPerBlobInfo is a rough estimation for the amount of extra bytes in
	// information a blob adds to the size of the underlying transaction.
	BytesPerBlobInfo = 70

	// DefaultGasPerBlobByte is the default gas cost deducted per byte of blob
	// included in a PayForBlobs txn
	DefaultGasPerBlobByte = 8

	DefaultTxSizeCostPerByte uint64 = 10
)

func (c *DataAvailabilityLayerClient) calculateFees(gas uint64) int64 {
	fees := c.config.Fee
	if fees == 0 {
		fees = int64(c.config.GasPrices * float64(gas))
	}

	return fees
}

// GasToConsume works out the extra gas charged to pay for a set of blobs in a PFB.
// Note that tranasctions will incur other gas costs, such as the signature verification
// and reads to the user's account.
func GasToConsume(blobSizes []uint32, gasPerByte uint32) uint64 {
	var totalSharesUsed uint64
	for _, size := range blobSizes {
		totalSharesUsed += uint64(types.SparseSharesNeeded(size))
	}

	return totalSharesUsed * types.ShareSize * uint64(gasPerByte)
}

// EstimateGas estimates the total gas required to pay for a set of blobs in a PFB.
// It is based on a linear model that is dependent on the governance parameters:
// gasPerByte and txSizeCost. It assumes other variables are constant. This includes
// assuming the PFB is the only message in the transaction.
func EstimateGas(blobSizes []uint32, gasPerByte uint32, txSizeCost uint64) uint64 {
	return GasToConsume(blobSizes, gasPerByte) + (txSizeCost * BytesPerBlobInfo * uint64(len(blobSizes))) + PFBGasFixedCost
}

// DefaultEstimateGas runs EstimateGas with the system defaults. The network may change these values
// through governance, thus this function should predominantly be used in testing.
func DefaultEstimateGas(blobSize uint32) uint64 {
	return EstimateGas([]uint32{blobSize}, DefaultGasPerBlobByte, DefaultTxSizeCostPerByte)
}
