package kaspa

import (
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/daemon/pb"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/util"
	"github.com/pkg/errors"
)

func (s *Client) refreshUTXOs() error {

	// No need to lock for reading since the only writer of this set is on `syncLoop` on the same goroutine.
	//addresses := s.addressSet.strings()
	addresses := s.addressSet.strings()
	// It's important to check the mempool before calling `GetUTXOsByAddresses`:
	// If we would do it the other way around an output can be spent in the mempool
	// and not in consensus, and between the calls its spending transaction will be
	// added to consensus and removed from the mempool, so `getUTXOsByAddressesResponse`
	// will include an obsolete output.
	mempoolEntriesByAddresses, err := s.rpcClient.GetMempoolEntriesByAddresses(addresses, true, true)
	if err != nil {
		return err
	}

	getUTXOsByAddressesResponse, err := s.rpcClient.GetUTXOsByAddresses(addresses)
	if err != nil {
		return err
	}

	return s.updateUTXOSet(getUTXOsByAddressesResponse.Entries, mempoolEntriesByAddresses.Entries)
}

// updateUTXOSet clears the current UTXO set, and re-fills it with the given entries
func (s *Client) updateUTXOSet(entries []*appmessage.UTXOsByAddressesEntry, mempoolEntries []*appmessage.MempoolEntryByAddress) error {
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

		// No need to lock for reading since the only writer of this set is on `syncLoop` on the same goroutine.
		address, ok := s.addressSet[entry.Address]
		if !ok {
			return errors.Errorf("Got result from address %s even though it wasn't requested", entry.Address)
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
	s.startTimeOfLastCompletedRefresh = time.Now()

	s.utxosSortedByAmount = utxos
	s.mempoolExcludedUTXOs = mempoolExcludedUTXOs

	// Cleanup expired used outpoints to avoid a memory leak
	for outpoint, broadcastTime := range s.usedOutpoints {
		if s.usedOutpointHasExpired(broadcastTime) {
			delete(s.usedOutpoints, outpoint)
		}
	}

	return nil
}

func (s *Client) usedOutpointHasExpired(outpointBroadcastTime time.Time) bool {
	// If the node returns a UTXO we previously attempted to spend and enough time has passed, we assume
	// that the network rejected or lost the previous transaction and allow a reuse. We set this time
	// interval to a minute.
	// We also verify that a full refresh UTXO operation started after this time point and has already
	// completed, in order to make sure that indeed this state reflects a state obtained following the required wait time.
	return s.startTimeOfLastCompletedRefresh.After(outpointBroadcastTime.Add(time.Minute))
}

func (s *Client) createUnsignedTransactions(address string, amount uint64, isSendAll bool, fromAddressesString []string, useExistingChangeAddress bool, requestFeePolicy *pb.FeePolicy) ([][]byte, error) {
	/*if !s.isSynced() {
		return nil, errors.Errorf("wallet daemon is not synced yet, %s", s.formatSyncStateReport())
	}*/

	feeRate, maxFee, err := s.calculateFeeLimits(requestFeePolicy)
	if err != nil {
		return nil, err
	}

	// make sure address string is correct before proceeding to a
	// potentially long UTXO refreshment operation
	toAddress, err := util.DecodeAddress(address, s.params.Prefix)
	if err != nil {
		return nil, err
	}

	var fromAddresses []*walletAddress
	for _, from := range fromAddressesString {
		fromAddress, exists := s.addressSet[from]
		if !exists {
			return nil, fmt.Errorf("specified from address %s does not exists", from)
		}
		fromAddresses = append(fromAddresses, fromAddress)
	}

	changeAddress, changeWalletAddress, err := s.changeAddress(useExistingChangeAddress, fromAddresses)
	if err != nil {
		return nil, err
	}

	selectedUTXOs, spendValue, changeSompi, err := s.selectUTXOs(amount, isSendAll, feeRate, maxFee, fromAddresses)
	if err != nil {
		return nil, err
	}

	if len(selectedUTXOs) == 0 {
		return nil, errors.Errorf("couldn't find funds to spend")
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
	unsignedTransaction, err := libkaspawallet.CreateUnsignedTransaction(s.keysFile.ExtendedPublicKeys,
		s.keysFile.MinimumSignatures,
		payments, selectedUTXOs)
	if err != nil {
		return nil, err
	}

	fmt.Println("utx", unsignedTransaction.Tx)
	unsignedTransactions, err := s.maybeAutoCompoundTransaction(unsignedTransaction, toAddress, changeAddress, changeWalletAddress, feeRate, maxFee)
	if err != nil {
		return nil, err
	}
	fmt.Println("utx2", len(unsignedTransactions[0]), hex.EncodeToString(unsignedTransactions[0]))

	return unsignedTransactions, nil
}

func (s *Client) selectUTXOs(spendAmount uint64, isSendAll bool, feeRate float64, maxFee uint64, fromAddresses []*walletAddress) (
	selectedUTXOs []*libkaspawallet.UTXO, totalReceived uint64, changeSompi uint64, err error) {
	return s.selectUTXOsWithPreselected(nil, map[externalapi.DomainOutpoint]struct{}{}, spendAmount, isSendAll, feeRate, maxFee, fromAddresses)
}

func (s *Client) selectUTXOsWithPreselected(preSelectedUTXOs []*walletUTXO, allowUsed map[externalapi.DomainOutpoint]struct{}, spendAmount uint64, isSendAll bool, feeRate float64, maxFee uint64, fromAddresses []*walletAddress) (
	selectedUTXOs []*libkaspawallet.UTXO, totalReceived uint64, changeSompi uint64, err error) {

	preSelectedSet := make(map[externalapi.DomainOutpoint]struct{})
	for _, utxo := range preSelectedUTXOs {
		preSelectedSet[*utxo.Outpoint] = struct{}{}
	}
	totalValue := uint64(0)

	dagInfo, err := s.rpcClient.GetBlockDAGInfo()
	if err != nil {
		return nil, 0, 0, err
	}

	var fee uint64
	iteration := func(utxo *walletUTXO, avoidPreselected bool) (bool, error) {
		if (fromAddresses != nil && !walletAddressesContain(fromAddresses, utxo.address)) ||
			!s.isUTXOSpendable(utxo, dagInfo.VirtualDAAScore) {
			return true, nil
		}

		if broadcastTime, ok := s.usedOutpoints[*utxo.Outpoint]; ok {
			if _, ok := allowUsed[*utxo.Outpoint]; !ok {
				if s.usedOutpointHasExpired(broadcastTime) {
					delete(s.usedOutpoints, *utxo.Outpoint)
				} else {
					return true, nil
				}
			}
		}

		if avoidPreselected {
			if _, ok := preSelectedSet[*utxo.Outpoint]; ok {
				return true, nil
			}
		}

		selectedUTXOs = append(selectedUTXOs, &libkaspawallet.UTXO{
			Outpoint:       utxo.Outpoint,
			UTXOEntry:      utxo.UTXOEntry,
			DerivationPath: s.walletAddressPath(utxo.address),
		})

		totalValue += utxo.UTXOEntry.Amount()
		estimatedRecipientValue := spendAmount
		if isSendAll {
			estimatedRecipientValue = totalValue
		}

		fee, err = s.estimateFee(selectedUTXOs, feeRate, maxFee, estimatedRecipientValue)
		if err != nil {
			return false, err
		}

		totalSpend := spendAmount + fee
		// Two break cases (if not send all):
		// 		1. totalValue == totalSpend, so there's no change needed -> number of outputs = 1, so a single input is sufficient
		// 		2. totalValue > totalSpend, so there will be change and 2 outputs, therefor in order to not struggle with --
		//		   2.1 go-nodes dust patch we try and find at least 2 inputs (even though the next one is not necessary in terms of spend value)
		// 		   2.2 KIP9 we try and make sure that the change amount is not too small
		if !isSendAll && (totalValue == totalSpend || (totalValue >= totalSpend+minChangeTarget && len(selectedUTXOs) > 1)) {
			return false, nil
		}

		return true, nil
	}

	shouldContinue := true
	for _, utxo := range preSelectedUTXOs {
		shouldContinue, err = iteration(utxo, false)
		if err != nil {
			return nil, 0, 0, err
		}

		if !shouldContinue {
			break
		}
	}

	if shouldContinue {
		for _, utxo := range s.utxosSortedByAmount {
			shouldContinue, err := iteration(utxo, true)
			if err != nil {
				return nil, 0, 0, err
			}

			if !shouldContinue {
				break
			}
		}
	}

	var totalSpend uint64
	if isSendAll {
		totalSpend = totalValue
		totalReceived = totalValue - fee
	} else {
		totalSpend = spendAmount + fee
		totalReceived = spendAmount
	}
	if totalValue < totalSpend {
		return nil, 0, 0, errors.Errorf("Insufficient funds for send: %f required, while only %f available",
			float64(totalSpend)/constants.SompiPerKaspa, float64(totalValue)/constants.SompiPerKaspa)
	}

	return selectedUTXOs, totalReceived, totalValue - totalSpend, nil
}
