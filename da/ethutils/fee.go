package ethutils

import (
	"fmt"
	"math/big"

	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/holiman/uint256"
)

// CalcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to:
//
//	gasTipCap + 2*baseFee.
func CalcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, big.NewInt(2)),
	)
}

// FinishBlobTx finishes creating a blob tx message by safely converting bigints to uint256
func FinishBlobTx(message *types.BlobTx, tip, fee, blobFee *big.Int) error {
	var o bool
	if message.GasTipCap, o = uint256.FromBig(tip); o {
		return fmt.Errorf("GasTipCap overflow")
	}
	if message.GasFeeCap, o = uint256.FromBig(fee); o {
		return fmt.Errorf("GasFeeCap overflow")
	}
	if message.BlobFeeCap, o = uint256.FromBig(blobFee); o {
		return fmt.Errorf("BlobFeeCap overflow")
	}
	return nil
}
