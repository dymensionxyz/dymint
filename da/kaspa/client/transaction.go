package client

import (
	"fmt"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/bip32"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"github.com/kaspanet/kaspad/domain/consensus/utils/utxo"
	"github.com/kaspanet/kaspad/domain/miningmanager/mempool"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/txmass"
)

func (c *Client) createUnsignedTransactions(utxos []*walletUTXO, blob []byte) ([][]byte, error) {
	feeRate, maxFee, err := c.calculateFeeLimits()
	if err != nil {
		return nil, err
	}

	selectedUTXOs, change, err := c.selectUTXOs(feeRate, maxFee, blob)
	if err != nil {
		return nil, err
	}

	if len(selectedUTXOs) == 0 {
		return nil, fmt.Errorf("couldn't find funds to spend")
	}

	payments := []*libkaspawallet.Payment{{
		Address: c.address,
		Amount:  change,
	}}
	publickey, err := c.extendedKey.Public()
	if err != nil {
		return nil, err
	}
	unsignedTransaction, err := createUnsignedTransaction(publickey.String(), payments, selectedUTXOs, blob)
	if err != nil {
		return nil, err
	}

	unsignedTransactions, err := c.maybeAutoCompoundTransaction(unsignedTransaction, c.address, utxos[0].address, feeRate, maxFee, blob)
	if err != nil {
		return nil, err
	}

	return unsignedTransactions, nil
}

// maybeAutoCompoundTransaction checks if a transaction's mass is higher that what is allowed for a standard
// transaction.
// If it is - the transaction is split into multiple transactions, each with a portion of the inputs and a single output
// into a change address.
// An additional `mergeTransaction` is generated - which merges the outputs of the above splits into a single output
// paying to the original transaction's payee.
func (c *Client) maybeAutoCompoundTransaction(transaction *serialization.PartiallySignedTransaction, address util.Address, wAddress *walletAddress,
	feeRate float64, maxFee uint64, blob []byte,
) ([][]byte, error) {
	splitTransactions, err := c.maybeSplitAndMergeTransaction(transaction, address, wAddress, feeRate, maxFee, blob)
	if err != nil {
		return nil, err
	}
	splitTransactionsBytes := make([][]byte, len(splitTransactions))
	for i, splitTransaction := range splitTransactions {
		splitTransactionsBytes[i], err = serialization.SerializePartiallySignedTransaction(splitTransaction)
		if err != nil {
			return nil, err
		}
	}
	return splitTransactionsBytes, nil
}

func (c *Client) maybeSplitAndMergeTransaction(transaction *serialization.PartiallySignedTransaction, address util.Address, wAddress *walletAddress,
	feeRate float64, maxFee uint64, blob []byte,
) ([]*serialization.PartiallySignedTransaction, error) {
	mockTx := transaction.Clone()
	mockTx.Tx.Payload = nil
	transactionMass, err := c.estimateComputeMassAfterSignatures(mockTx)
	if err != nil {
		return nil, err
	}
	transientMass, err := c.estimateTransientMassAfterSignatures(transaction)
	if err != nil {
		return nil, err
	}
	if transientMass < mempool.MaximumStandardTransactionMass {
		return []*serialization.PartiallySignedTransaction{transaction}, nil
	} else {

		maxChunkSize := (int)((mempool.MaximumStandardTransactionMass - transactionMass) / TRANSIENT_BYTE_TO_MASS_FACTOR) //nolint:gosec // maxChunkSize will not overflow
		splitCount := (len(blob) / maxChunkSize)

		if len(blob)%maxChunkSize > 0 {
			splitCount++
		}
		splitTransactions := make([]*serialization.PartiallySignedTransaction, splitCount)

		for i := 0; i < splitCount; i++ {
			totalSompi := uint64(0)
			startChunkIndex := i * maxChunkSize
			endChunkIndex := startChunkIndex + maxChunkSize
			if endChunkIndex > len(blob)-1 {
				endChunkIndex = len(blob)
			}
			var selectedUTXOs []*libkaspawallet.UTXO
			if i == 0 {
				partiallySignedInput := transaction.PartiallySignedInputs[0]
				selectedUTXOs = append(selectedUTXOs, &libkaspawallet.UTXO{
					Outpoint: &transaction.Tx.Inputs[0].PreviousOutpoint,
					UTXOEntry: utxo.NewUTXOEntry(
						partiallySignedInput.PrevOutput.Value, partiallySignedInput.PrevOutput.ScriptPublicKey,
						false, constants.UnacceptedDAAScore),
					DerivationPath: partiallySignedInput.DerivationPath,
				})
			} else {
				output := splitTransactions[i-1].Tx.Outputs[0]
				selectedUTXOs = append(selectedUTXOs, &libkaspawallet.UTXO{
					Outpoint: &externalapi.DomainOutpoint{
						TransactionID: *consensushashing.TransactionID(splitTransactions[i-1].Tx),
						Index:         0,
					},
					UTXOEntry:      utxo.NewUTXOEntry(output.Value, output.ScriptPublicKey, false, constants.UnacceptedDAAScore),
					DerivationPath: walletAddressPath(wAddress),
				})
			}

			totalSompi += selectedUTXOs[0].UTXOEntry.Amount()
			fee, err := c.estimateFee(selectedUTXOs, feeRate, maxFee, blob[startChunkIndex:endChunkIndex])
			if err != nil {
				return nil, err
			}

			totalSompi -= fee
			publickey, err := c.extendedKey.Public()
			if err != nil {
				return nil, err
			}
			payload := blob[startChunkIndex:endChunkIndex]
			tx, err := createUnsignedTransaction(publickey.String(),
				[]*libkaspawallet.Payment{{
					Address: address,
					Amount:  totalSompi,
				}}, selectedUTXOs, payload)
			if err != nil {
				return nil, err
			}
			splitTransactions[i] = tx
		}

		return splitTransactions, nil
	}
}

func createUnsignedTransaction(
	extendedPublicKey string,
	payments []*libkaspawallet.Payment,
	selectedUTXOs []*libkaspawallet.UTXO,
	blob []byte,
) (*serialization.PartiallySignedTransaction, error) {
	inputs := make([]*externalapi.DomainTransactionInput, len(selectedUTXOs))
	partiallySignedInputs := make([]*serialization.PartiallySignedInput, len(selectedUTXOs))

	for i, utxo := range selectedUTXOs {

		extendedKey, err := bip32.DeserializeExtendedKey(extendedPublicKey)
		if err != nil {
			return nil, err
		}

		derivedKey, err := extendedKey.DeriveFromPath(utxo.DerivationPath)
		if err != nil {
			return nil, err
		}
		emptyPubKeySignaturePair := []*serialization.PubKeySignaturePair{
			{
				ExtendedPublicKey: derivedKey.String(),
			},
		}

		inputs[i] = &externalapi.DomainTransactionInput{PreviousOutpoint: *utxo.Outpoint}

		partiallySignedInputs[i] = &serialization.PartiallySignedInput{
			PrevOutput: &externalapi.DomainTransactionOutput{
				Value:           utxo.UTXOEntry.Amount(),
				ScriptPublicKey: utxo.UTXOEntry.ScriptPublicKey(),
			},
			MinimumSignatures:    1,
			PubKeySignaturePairs: emptyPubKeySignaturePair,
			DerivationPath:       utxo.DerivationPath,
		}
	}

	outputs := make([]*externalapi.DomainTransactionOutput, len(payments))
	for i, payment := range payments {
		scriptPublicKey, err := txscript.PayToAddrScript(payment.Address)
		if err != nil {
			return nil, err
		}

		outputs[i] = &externalapi.DomainTransactionOutput{
			Value:           payment.Amount,
			ScriptPublicKey: scriptPublicKey,
		}
	}

	domainTransaction := &externalapi.DomainTransaction{
		Version:      constants.MaxTransactionVersion,
		Inputs:       inputs,
		Outputs:      outputs,
		LockTime:     0,
		SubnetworkID: subnetworks.SubnetworkIDNative,
		Gas:          0,
		Payload:      blob,
	}

	return &serialization.PartiallySignedTransaction{
		Tx:                    domainTransaction,
		PartiallySignedInputs: partiallySignedInputs,
	}, nil
}

func createTransactionWithJunkFieldsForMassCalculation(transaction *serialization.PartiallySignedTransaction, ecdsa bool, minimumSignatures uint32, txMassCalculator *txmass.Calculator) (*externalapi.DomainTransaction, error) {
	transaction = transaction.Clone()
	signatureSize := uint64(64)

	for i, input := range transaction.PartiallySignedInputs {
		for j, pubKeyPair := range input.PubKeySignaturePairs {
			if uint32(j) >= minimumSignatures { //nolint:gosec // input.PubKeySignaturePairs number cannot overflow
				break
			}
			pubKeyPair.Signature = make([]byte, signatureSize+1) // +1 for SigHashType
		}
		transaction.Tx.Inputs[i].SigOpCount = byte(len(input.PubKeySignaturePairs))
	}

	return libkaspawallet.ExtractTransactionDeserialized(transaction, ecdsa)
}
