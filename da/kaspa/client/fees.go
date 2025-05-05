package client

import (
	"math"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/txmass"
)

// estimate overall tx mass (max between storage and compute mass)
func (c *Client) estimateMassAfterSignatures(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	return estimateMassAfterSignatures(transaction, false, 1, c.txMassCalculator)
}

// estimate transient mass (max tx size used to accept tx in mempool)
func (c *Client) estimateTransientMassAfterSignatures(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	return estimateTransientMass(transaction)
}

// isUTXOSpendable returns true if the utxo is not coinbase type (from miners) and the utxos are old enough based on the DAAscore
func (c *Client) isUTXOSpendable(entry *walletUTXO, virtualDAAScore uint64) bool {
	if !entry.UTXOEntry.IsCoinbase() {
		return true
	}
	return entry.UTXOEntry.BlockDAAScore()+c.params.BlockCoinbaseMaturity < virtualDAAScore
}

// calculateFeeLimits fee rate and max fee used for fee estimation
func (c *Client) calculateFeeLimits() (feeRate float64, maxFee uint64, err error) {
	estimate, err := c.rpcClient.GetFeeEstimate()
	if err != nil {
		return 0, 0, err
	}
	feeRate = estimate.Estimate.NormalBuckets[0].Feerate
	// Default to a bound of max 1 KAS as fee
	maxFee = constants.MaxSompi

	return feeRate, maxFee, nil
}

// estimateFee estimates the necessary fee to send a tx
func (c *Client) estimateFee(selectedUTXOs []*libkaspawallet.UTXO, feeRate float64, maxFee uint64, blob []byte) (uint64, error) {
	fakePubKey := [util.PublicKeySizeECDSA]byte{}
	fakeAddr, err := util.NewAddressPublicKeyECDSA(fakePubKey[:], c.params.Prefix) // We assume the worst case where the recipient address is ECDSA. In this case the scriptPubKey will be the longest.
	if err != nil {
		return 0, err
	}

	totalValue := uint64(0)
	for _, utxo := range selectedUTXOs {
		totalValue += utxo.UTXOEntry.Amount()
	}

	// This is an approximation for the distribution of value between the recipient output and the change output.
	mockPayments := []*libkaspawallet.Payment{
		{
			Address: fakeAddr,
			Amount:  totalValue,
		},
	}

	mockTx, err := createUnsignedTransaction(c.publicKey.String(), mockPayments, selectedUTXOs, blob)
	if err != nil {
		return 0, err
	}

	mass, err := c.estimateMassAfterSignatures(mockTx)
	if err != nil {
		return 0, err
	}

	return min(uint64(math.Ceil(float64(mass)*feeRate)), maxFee), nil
}

// estimateTransientMass calculates the transient mass, used to accept in mempool based only on size (see KIP-0013)
func estimateTransientMass(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	serializedTx, err := serialization.SerializePartiallySignedTransaction(transaction)
	if err != nil {
		return uint64(0), err
	}
	return uint64(len(serializedTx) * TRANSIENT_BYTE_TO_MASS_FACTOR), nil //nolint:gosec // serializedTx size will not overflow
}

// estimateTransientMass calculates the overall mass of the transaction including compute and storage mass components (see KIP-0009)
func estimateMassAfterSignatures(transaction *serialization.PartiallySignedTransaction, ecdsa bool, minimumSignatures uint32, txMassCalculator *txmass.Calculator) (uint64, error) {
	transactionWithSignatures, err := createTransactionWithJunkFieldsForMassCalculation(transaction, ecdsa, minimumSignatures, txMassCalculator)
	if err != nil {
		return 0, err
	}

	return txMassCalculator.CalculateTransactionOverallMass(transactionWithSignatures), nil
}
