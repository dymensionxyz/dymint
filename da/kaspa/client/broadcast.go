package client

import (
	"cosmossdk.io/errors"
	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/consensushashing"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
)

// broadcast sends signed txs to the Kaspa node, and returns tx ids if not rejected.
func (c *Client) broadcast(transactions [][]byte) ([]string, error) {
	txIDs := make([]string, len(transactions))
	var tx *externalapi.DomainTransaction
	var err error

	for i, transaction := range transactions {
		tx, err = libkaspawallet.ExtractTransaction(transaction, false)
		if err != nil {
			return nil, err
		}
		txIDs[i], err = sendTransaction(c.rpcClient, tx)
		if err != nil {
			return nil, err
		}

	}

	return txIDs, nil
}

// sendTransaction sends a grpc message to the Kaspa node including a tx, and returns the result received from the node
func sendTransaction(client *rpcclient.RPCClient, tx *externalapi.DomainTransaction) (string, error) {
	submitTransactionResponse, err := client.SubmitTransaction(appmessage.DomainTransactionToRPCTransaction(tx), consensushashing.TransactionID(tx).String(), false)
	if err != nil {
		return "", errors.Wrapf(err, "error submitting transaction")
	}
	return submitTransactionResponse.TransactionID, nil
}
