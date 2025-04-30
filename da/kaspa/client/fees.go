package client

import (
	"math"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/txmass"
)

func (c *Client) estimateMassAfterSignatures(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	return EstimateMassAfterSignatures(transaction, false, 1, c.txMassCalculator)
}

func (c *Client) estimateTransientMassAfterSignatures(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	return estimateTransientMass(transaction)
}

func (c *Client) estimateComputeMassAfterSignatures(transaction *serialization.PartiallySignedTransaction) (uint64, error) {
	return estimateComputeMassAfterSignatures(transaction, false, 1, c.txMassCalculator)
}

func createTransactionWithJunkFieldsForMassCalculation(transaction *serialization.PartiallySignedTransaction, ecdsa bool, minimumSignatures uint32, txMassCalculator *txmass.Calculator) (*externalapi.DomainTransaction, error) {
	transaction = transaction.Clone()
	signatureSize := uint64(64)

	for i, input := range transaction.PartiallySignedInputs {
		for j, pubKeyPair := range input.PubKeySignaturePairs {
			if uint32(j) >= minimumSignatures {
				break
			}
			pubKeyPair.Signature = make([]byte, signatureSize+1) // +1 for SigHashType
		}
		transaction.Tx.Inputs[i].SigOpCount = byte(len(input.PubKeySignaturePairs))
	}

	return libkaspawallet.ExtractTransactionDeserialized(transaction, ecdsa)
}

func estimateComputeMassAfterSignatures(transaction *serialization.PartiallySignedTransaction, ecdsa bool, minimumSignatures uint32, txMassCalculator *txmass.Calculator) (uint64, error) {
	transactionWithSignatures, err := createTransactionWithJunkFieldsForMassCalculation(transaction, ecdsa, minimumSignatures, txMassCalculator)
	if err != nil {
		return 0, err
	}

	return txMassCalculator.CalculateTransactionMass(transactionWithSignatures), nil
}

func estimateTransientMass(transaction *serialization.PartiallySignedTransaction) (uint64, error) {

	serializedTx, err := serialization.SerializePartiallySignedTransaction(transaction)
	if err != nil {
		return uint64(0), err
	}
	return uint64(len(serializedTx) * TRANSIENT_BYTE_TO_MASS_FACTOR), nil
}

func EstimateMassAfterSignatures(transaction *serialization.PartiallySignedTransaction, ecdsa bool, minimumSignatures uint32, txMassCalculator *txmass.Calculator) (uint64, error) {
	transactionWithSignatures, err := createTransactionWithJunkFieldsForMassCalculation(transaction, ecdsa, minimumSignatures, txMassCalculator)
	if err != nil {
		return 0, err
	}

	return txMassCalculator.CalculateTransactionOverallMass(transactionWithSignatures), nil
}

func (c *Client) isUTXOSpendable(entry *walletUTXO, virtualDAAScore uint64) bool {
	if !entry.UTXOEntry.IsCoinbase() {
		return true
	}
	return entry.UTXOEntry.BlockDAAScore()+c.coinbaseMaturity < virtualDAAScore
}

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

func (c *Client) estimateFee(selectedUTXOs []*libkaspawallet.UTXO, feeRate float64, maxFee uint64, recipientValue uint64, blob []byte) (uint64, error) {
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
	var mockPayments []*libkaspawallet.Payment
	if totalValue > recipientValue {
		mockPayments = []*libkaspawallet.Payment{
			{
				Address: fakeAddr,
				Amount:  recipientValue,
			},
			{
				Address: fakeAddr,
				Amount:  totalValue - recipientValue, // We ignore the fee since we expect it to be insignificant in mass calculation.
			},
		}
	} else {
		mockPayments = []*libkaspawallet.Payment{
			{
				Address: fakeAddr,
				Amount:  totalValue,
			},
		}
	}

	publickey, err := c.extendedKey.Public()
	if err != nil {
		return 0, err
	}

	mockTx, err := createUnsignedTransaction(publickey.String(), mockPayments, selectedUTXOs, blob)
	if err != nil {
		return 0, err
	}

	mass, err := c.estimateMassAfterSignatures(mockTx)
	if err != nil {
		return 0, err
	}

	return min(uint64(math.Ceil(float64(mass)*feeRate)), maxFee), nil
}
