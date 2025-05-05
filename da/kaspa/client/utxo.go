package client

import (
	"fmt"
	"sort"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
)

type walletUTXO struct {
	Outpoint  *externalapi.DomainOutpoint
	UTXOEntry externalapi.UTXOEntry
	address   *walletAddress
}

// getUTXOs retrieves spendable Utxos per address from Kaspa node
func (c *Client) getUTXOs() ([]*walletUTXO, error) {
	// It's important to check the mempool before calling `GetUTXOsByAddresses`:
	// If we would do it the other way around an output can be spent in the mempool
	// and not in consensus, and between the calls its spending transaction will be
	// added to consensus and removed from the mempool, so `getUTXOsByAddressesResponse`
	// will include an obsolete output.
	mempoolEntriesByAddresses, err := c.rpcClient.GetMempoolEntriesByAddresses([]string{c.address.String()}, true, true)
	if err != nil {
		return nil, err
	}

	getUTXOsByAddressesResponse, err := c.rpcClient.GetUTXOsByAddresses([]string{c.address.String()})
	if err != nil {
		return nil, err
	}

	return c.generateWalletUtxos(getUTXOsByAddressesResponse.Entries, mempoolEntriesByAddresses.Entries)
}

// generateWalletUtxos translates entries received from Kaspa Node to walletUTXO
func (c *Client) generateWalletUtxos(entries []*appmessage.UTXOsByAddressesEntry, mempoolEntries []*appmessage.MempoolEntryByAddress) ([]*walletUTXO, error) {
	utxos := make([]*walletUTXO, 0, len(entries))

	exclude := make(map[appmessage.RPCOutpoint]struct{})
	for _, entriesByAddress := range mempoolEntries {
		for _, entry := range entriesByAddress.Sending {
			for _, input := range entry.Transaction.Inputs {
				exclude[*input.PreviousOutpoint] = struct{}{}
			}
		}
	}

	// we obtain walletAddress corresponding to the configuration address, by generating the addresses corresponding to different index (max 100) and stopping once found
	var address *walletAddress
	for index := uint32(0); index < maxAddressesUtxo; index++ {
		candidateAddress := &walletAddress{
			index:         index,
			cosignerIndex: 0,
			keyChain:      libkaspawallet.ExternalKeychain,
		}
		addressString, err := c.walletAddressString(candidateAddress)
		if err != nil {
			return nil, err
		}
		if addressString == c.address.String() {
			address = candidateAddress
			break
		}
	}
	if address == nil {
		return nil, fmt.Errorf("address not found")
	}

	mempoolExcludedUTXOs := make(map[externalapi.DomainOutpoint]*walletUTXO)
	for _, entry := range entries {
		outpoint, err := appmessage.RPCOutpointToDomainOutpoint(entry.Outpoint)
		if err != nil {
			return nil, err
		}

		utxoEntry, err := appmessage.RPCUTXOEntryToUTXOEntry(entry.UTXOEntry)
		if err != nil {
			return nil, err
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

	return utxos, nil
}

// selectUTXOs, returns Utxos to be used, necessary to send tx. Returns error in case not enough funds.
func (c *Client) selectUTXOs(feeRate float64, maxFee uint64, blob []byte) (selectedUTXOs []*libkaspawallet.UTXO, changeSompi uint64, err error) {
	utxos, err := c.getUTXOs()
	if err != nil {
		return nil, 0, err
	}
	c.wAddress = utxos[0].address
	totalValue := uint64(0)

	dagInfo, err := c.rpcClient.GetBlockDAGInfo()
	if err != nil {
		return nil, 0, err
	}

	var fee uint64

	iteration := func(utxo *walletUTXO) (bool, error) {
		if !c.isUTXOSpendable(utxo, dagInfo.VirtualDAAScore) {
			return true, nil
		}

		selectedUTXOs = append(selectedUTXOs, &libkaspawallet.UTXO{
			Outpoint:       utxo.Outpoint,
			UTXOEntry:      utxo.UTXOEntry,
			DerivationPath: walletAddressPath(utxo.address),
		})

		totalValue += utxo.UTXOEntry.Amount()

		fee, err = c.estimateFee(selectedUTXOs, feeRate, maxFee, blob)
		if err != nil {
			return false, err
		}

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

	for _, utxo := range utxos {
		shouldContinue, err := iteration(utxo)
		if err != nil {
			return nil, 0, err
		}

		if !shouldContinue {
			break
		}
	}

	totalSpend := fee
	// totalReceived = spendAmount
	if totalValue < totalSpend {
		return nil, 0, fmt.Errorf("Insufficient funds for send: %f required, while only %f available",
			float64(totalSpend)/constants.SompiPerKaspa, float64(totalValue)/constants.SompiPerKaspa)
	}

	return selectedUTXOs, totalValue - totalSpend, nil
}
