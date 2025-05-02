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

// generateBlobTxs returns serialized Kaspa txs that includes the blob in the payload
func (c *Client) generateBlobTxs(blob []byte) ([][]byte, error) {
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

	unsignedTransaction, err := createUnsignedTransaction(c.publicKey.String(), payments, selectedUTXOs, blob)
	if err != nil {
		return nil, err
	}

	unsignedTransactions, err := c.serializeTransaction(unsignedTransaction, c.address, c.wAddress, feeRate, maxFee, blob)
	if err != nil {
		return nil, err
	}

	return unsignedTransactions, nil
}

// serializeTransaction splits the transaction, when necessary, into multiple depending on the mass and serializes them into []byte
func (c *Client) serializeTransaction(transaction *serialization.PartiallySignedTransaction, address util.Address, wAddress *walletAddress,
	feeRate float64, maxFee uint64, blob []byte,
) ([][]byte, error) {
	splitTransactions, err := c.calculateMassAndSplitTransaction(transaction, address, wAddress, feeRate, maxFee, blob)
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

// calculateMassAndSplitTransaction splits the transaction into multiple in case transientMass is above the mempool limit (maximum mass accepted by nodes, otherwise they reject the tx before added to mempool).
func (c *Client) calculateMassAndSplitTransaction(transaction *serialization.PartiallySignedTransaction, address util.Address, wAddress *walletAddress,
	feeRate float64, maxFee uint64, blob []byte,
) ([]*serialization.PartiallySignedTransaction, error) {

	transientMass, err := c.estimateTransientMassAfterSignatures(transaction)
	if err != nil {
		return nil, err
	}

	// we assume transientMass is always higher than overall transaction mass, which will be the case when including payload with small number of inputs.
	if transientMass < mempool.MaximumStandardTransactionMass {
		return []*serialization.PartiallySignedTransaction{transaction}, nil
	} else {
		mockTx := transaction.Clone()
		mockTx.Tx.Payload = nil

		seralizedTx, err := serialization.SerializePartiallySignedTransaction(mockTx)
		if err != nil {
			return nil, err
		}

		// this calculated the number of txs necessary and creates them (based on available size for the payload), chunking the blob and including a part of it into each tx sequentially
		maxChunkSize := (mempool.MaximumStandardTransactionMass / TRANSIENT_BYTE_TO_MASS_FACTOR) - len(seralizedTx)
		splitCount := len(blob) / maxChunkSize

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

			payload := blob[startChunkIndex:endChunkIndex]
			tx, err := createUnsignedTransaction(c.publicKey.String(),
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

// createUnsignedTransaction generates the transaction, using the selectedUTXOs as input, and generating outputs including the specified payments.
// The payment basically subtracts the fee from the initial Utxo value and sends the change to the same address.
// The blob is included to the tx as payload.
func createUnsignedTransaction(
	extendedPublicKey string,
	payments []*libkaspawallet.Payment,
	selectedUTXOs []*libkaspawallet.UTXO,
	blob []byte,
) (*serialization.PartiallySignedTransaction, error) {
	inputs := make([]*externalapi.DomainTransactionInput, len(selectedUTXOs))
	partiallySignedInputs := make([]*serialization.PartiallySignedInput, len(selectedUTXOs))

	for i, utxo := range selectedUTXOs {

		//TODO: verify this part of the code (copied from kaspawallet https://github.com/kaspanet/kaspad). It may be redundant
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

// createTransactionWithJunkFieldsForMassCalculation creates tx used only for estimating fees
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
