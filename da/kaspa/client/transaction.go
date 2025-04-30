package client

import (
	"fmt"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/bip32"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/serialization"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/consensus/utils/subnetworks"
	"github.com/kaspanet/kaspad/domain/consensus/utils/txscript"
	"github.com/kaspanet/kaspad/domain/miningmanager/mempool"
	"github.com/kaspanet/kaspad/util"
)

func (c *Client) createUnsignedTransactions(utxos []*walletUTXO, address string, blob []byte) ([][]byte, error) {

	feeRate, maxFee, err := c.calculateFeeLimits()
	if err != nil {
		return nil, err
	}

	// make sure address string is correct before proceeding to a
	// potentially long UTXO refreshment operation
	toAddress, err := util.DecodeAddress(address, c.params.Prefix)
	if err != nil {
		return nil, err
	}

	var fromAddresses []*walletAddress

	changeAddress, err := util.DecodeAddress(c.fromAddress, c.params.Prefix)
	if err != nil {
		return nil, err
	}

	selectedUTXOs, spendValue, changeSompi, err := c.selectUTXOs(utxos, feeRate, maxFee, fromAddresses, blob)
	if err != nil {
		return nil, err
	}

	if len(selectedUTXOs) == 0 {
		return nil, fmt.Errorf("couldn't find funds to spend")
	}

	payments := []*libkaspawallet.Payment{{
		Address: toAddress,
		Amount:  spendValue,
	}}
	if changeSompi > 0 {
		payments = append(payments, &libkaspawallet.Payment{
			Address: changeAddress,
			Amount:  changeSompi,
		})
	}

	publickey, err := c.extendedKey.Public()
	if err != nil {
		return nil, err
	}
	unsignedTransaction, err := createUnsignedTransaction(publickey.String(), payments, selectedUTXOs, blob)
	if err != nil {
		return nil, err
	}

	unsignedTransactions, err := c.maybeAutoCompoundTransaction(unsignedTransaction, toAddress, nil, nil, feeRate, maxFee, blob)
	if err != nil {
		return nil, err
	}

	return unsignedTransactions, nil
}

func createUnsignedTransaction(
	extendedPublicKey string,
	payments []*libkaspawallet.Payment,
	selectedUTXOs []*libkaspawallet.UTXO,
	blob []byte) (*serialization.PartiallySignedTransaction, error) {

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
			&serialization.PubKeySignaturePair{
				ExtendedPublicKey: derivedKey.String()},
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

// maybeAutoCompoundTransaction checks if a transaction's mass is higher that what is allowed for a standard
// transaction.
// If it is - the transaction is split into multiple transactions, each with a portion of the inputs and a single output
// into a change address.
// An additional `mergeTransaction` is generated - which merges the outputs of the above splits into a single output
// paying to the original transaction's payee.
func (s *Client) maybeAutoCompoundTransaction(transaction *serialization.PartiallySignedTransaction, toAddress util.Address,
	changeAddress util.Address, changeWalletAddress *walletAddress, feeRate float64, maxFee uint64, blob []byte) ([][]byte, error) {

	splitTransactions, err := s.maybeSplitAndMergeTransaction(transaction, toAddress, changeAddress, changeWalletAddress, feeRate, maxFee, blob)
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

func (s *Client) maybeSplitAndMergeTransaction(transaction *serialization.PartiallySignedTransaction, toAddress util.Address,
	changeAddress util.Address, changeWalletAddress *walletAddress, feeRate float64, maxFee uint64, blob []byte) ([]*serialization.PartiallySignedTransaction, error) {

	transactionMass, err := s.estimateComputeMassAfterSignatures(transaction)
	if err != nil {
		return nil, err
	}
	transientMass, err := s.estimateTransientMassAfterSignatures(transaction)

	if max(transientMass, transactionMass) < mempool.MaximumStandardTransactionMass {
		return []*serialization.PartiallySignedTransaction{transaction}, nil
	} else {
		return nil, fmt.Errorf("transaction mass to high. Max:%d,TransientMass:%d TransactionMass: %d", mempool.MaximumStandardTransactionMass, transientMass, transactionMass)
	}
}
