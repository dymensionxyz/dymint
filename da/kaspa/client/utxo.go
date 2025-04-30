package client

import (
	"sort"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/utils"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/pkg/errors"
)

type walletUTXO struct {
	Outpoint  *externalapi.DomainOutpoint
	UTXOEntry externalapi.UTXOEntry
	address   *walletAddress
}

func (c *Client) refreshUTXOs() error {

	// No need to lock for reading since the only writer of this set is on `syncLoop` on the same goroutine.
	//addresses := s.addressSet.strings()
	addresses := []string{fromAddress}
	// It's important to check the mempool before calling `GetUTXOsByAddresses`:
	// If we would do it the other way around an output can be spent in the mempool
	// and not in consensus, and between the calls its spending transaction will be
	// added to consensus and removed from the mempool, so `getUTXOsByAddressesResponse`
	// will include an obsolete output.
	mempoolEntriesByAddresses, err := c.rpcClient.GetMempoolEntriesByAddresses(addresses, true, true)
	if err != nil {
		return err
	}

	getUTXOsByAddressesResponse, err := c.rpcClient.GetUTXOsByAddresses(addresses)
	if err != nil {
		return err
	}

	return c.updateUTXOSet(getUTXOsByAddressesResponse.Entries, mempoolEntriesByAddresses.Entries)
}

// updateUTXOSet clears the current UTXO set, and re-fills it with the given entries
func (c *Client) updateUTXOSet(entries []*appmessage.UTXOsByAddressesEntry, mempoolEntries []*appmessage.MempoolEntryByAddress) error {
	utxos := make([]*walletUTXO, 0, len(entries))

	exclude := make(map[appmessage.RPCOutpoint]struct{})
	for _, entriesByAddress := range mempoolEntries {
		for _, entry := range entriesByAddress.Sending {
			for _, input := range entry.Transaction.Inputs {
				exclude[*input.PreviousOutpoint] = struct{}{}
			}
		}
	}

	mempoolExcludedUTXOs := make(map[externalapi.DomainOutpoint]*walletUTXO)
	for _, entry := range entries {
		outpoint, err := appmessage.RPCOutpointToDomainOutpoint(entry.Outpoint)
		if err != nil {
			return err
		}

		utxoEntry, err := appmessage.RPCUTXOEntryToUTXOEntry(entry.UTXOEntry)
		if err != nil {
			return err
		}

		address := &walletAddress{
			index:         1,
			cosignerIndex: 0,
			keyChain:      libkaspawallet.ExternalKeychain,
		}
		utxo := &walletUTXO{
			Outpoint:  outpoint,
			UTXOEntry: utxoEntry,
			address:   address,
		}

		if _, ok := exclude[*entry.Outpoint]; ok {
			mempoolExcludedUTXOs[*outpoint] = utxo
		} else {
			utxos = append(utxos, &walletUTXO{
				Outpoint:  outpoint,
				UTXOEntry: utxoEntry,
				address:   address,
			})
		}
	}

	sort.Slice(utxos, func(i, j int) bool { return utxos[i].UTXOEntry.Amount() > utxos[j].UTXOEntry.Amount() })

	c.utxosSortedByAmount = utxos
	c.mempoolExcludedUTXOs = mempoolExcludedUTXOs

	return nil
}

func (s *Client) selectUTXOs(feeRate float64, maxFee uint64, fromAddresses []*walletAddress, blob []byte) (selectedUTXOs []*libkaspawallet.UTXO, totalReceived uint64, changeSompi uint64, err error) {

	spendAmount, err := utils.KasToSompi("1")
	if err != nil {
		return nil, 0, 0, err
	}

	totalValue := uint64(0)

	dagInfo, err := s.rpcClient.GetBlockDAGInfo()
	if err != nil {
		return nil, 0, 0, err
	}

	var fee uint64
	iteration := func(utxo *walletUTXO) (bool, error) {
		if (fromAddresses != nil && !walletAddressesContain(fromAddresses, utxo.address)) ||
			!s.isUTXOSpendable(utxo, dagInfo.VirtualDAAScore) {
			return true, nil
		}

		selectedUTXOs = append(selectedUTXOs, &libkaspawallet.UTXO{
			Outpoint:       utxo.Outpoint,
			UTXOEntry:      utxo.UTXOEntry,
			DerivationPath: walletAddressPath(utxo.address),
		})

		totalValue += utxo.UTXOEntry.Amount()
		estimatedRecipientValue := spendAmount

		fee, err = s.estimateFee(selectedUTXOs, feeRate, maxFee, estimatedRecipientValue, blob)
		if err != nil {
			return false, err
		}

		//totalSpend := spendAmount + fee
		totalSpend := fee
		// Two break cases (if not send all):
		// 		1. totalValue == totalSpend, so there's no change needed -> number of outputs = 1, so a single input is sufficient
		// 		2. totalValue > totalSpend, so there will be change and 2 outputs, therefor in order to not struggle with --
		//		   2.1 go-nodes dust patch we try and find at least 2 inputs (even though the next one is not necessary in terms of spend value)
		// 		   2.2 KIP9 we try and make sure that the change amount is not too small
		if totalValue == totalSpend || (totalValue >= totalSpend+minChangeTarget && len(selectedUTXOs) > 1) {
			return false, nil
		}

		return true, nil
	}

	for _, utxo := range s.utxosSortedByAmount {
		shouldContinue, err := iteration(utxo)
		if err != nil {
			return nil, 0, 0, err
		}

		if !shouldContinue {
			break
		}
	}
	//}

	var totalSpend uint64
	totalSpend = spendAmount + fee
	totalReceived = spendAmount
	if totalValue < totalSpend {
		return nil, 0, 0, errors.Errorf("Insufficient funds for send: %f required, while only %f available",
			float64(totalSpend)/constants.SompiPerKaspa, float64(totalValue)/constants.SompiPerKaspa)
	}

	return selectedUTXOs, totalReceived, totalValue - totalSpend, nil
}
