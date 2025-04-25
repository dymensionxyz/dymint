package kaspa

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/keys"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/utils"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/kaspanet/kaspad/util/txmass"
)

const (
	keysPath                            = "/Users/sergi/Library/Application Support/Kaspawallet/kaspa-testnet-10/keys.json"
	minChangeTarget                     = constants.SompiPerKaspa * 10
	minFeeRate                          = 1.0
	address                             = "kaspatest:qp96y8xa6gqlh3a5c6wu9x73a5egvsw2vk7w7nzm8x98wvkavjlg29zvta4m6"
	numIndexesToQueryForFarAddresses    = 100
	numIndexesToQueryForRecentAddresses = 1000
)

type KaspaClient interface {
	SubmitBlob(blob []byte) (string, error)
	GetBlob(txHash string) ([]byte, error)
	Disconnect()
}

type walletUTXO struct {
	Outpoint  *externalapi.DomainOutpoint
	UTXOEntry externalapi.UTXOEntry
	address   *walletAddress
}

type walletAddress struct {
	index         uint32
	cosignerIndex uint32
	keyChain      uint8
}

type balancesType struct{ available, pending uint64 }
type balancesMapType map[*walletAddress]*balancesType

type walletAddressSet map[string]*walletAddress

func (was walletAddressSet) strings() []string {
	addresses := make([]string, 0, len(was))
	for addr := range was {
		addresses = append(addresses, addr)
	}
	return addresses
}

type Client struct {
	rpcClient                       *rpcclient.RPCClient // RPC client for ongoing user requests
	httpClient                      *http.Client
	params                          *dagconfig.Params
	coinbaseMaturity                uint64 // Is different from default if we use testnet-11
	usedOutpoints                   map[externalapi.DomainOutpoint]time.Time
	startTimeOfLastCompletedRefresh time.Time

	utxosSortedByAmount  []*walletUTXO
	keysFile             *keys.File
	txMassCalculator     *txmass.Calculator
	mempoolExcludedUTXOs map[externalapi.DomainOutpoint]*walletUTXO
	addressSet           walletAddressSet
	nextSyncStartIndex   uint32
}

var _ KaspaClient = &Client{}

func NewClient(ctx context.Context, config *Config) (KaspaClient, error) {
	rpcClient, err := rpcclient.NewRPCClient(config.GrpcAddress)
	if err != nil {
		return nil, err
	}

	if config.Timeout != 0 {
		rpcClient.SetTimeout(time.Duration(config.Timeout) * time.Second)
	}

	keysFile, err := keys.ReadKeysFile(&dagconfig.Params{}, keysPath)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	kaspaClient := &Client{
		rpcClient:          rpcClient,
		httpClient:         httpClient,
		coinbaseMaturity:   100,
		keysFile:           keysFile,
		params:             &dagconfig.TestnetParams,
		nextSyncStartIndex: 0,
		addressSet:         make(walletAddressSet),
		txMassCalculator:   txmass.NewCalculator(1, 10, 1000),
		usedOutpoints:      map[externalapi.DomainOutpoint]time.Time{},
	}
	return kaspaClient, nil

}

func (c *Client) Disconnect() {
	c.rpcClient.Disconnect()
}

func (c *Client) SubmitBlob(blob []byte) (string, error) {

	sendAmountSompi, err := utils.KasToSompi("0")
	if err != nil {
		return "", err
	}

	err = c.collectFarAddresses()
	if err != nil {
		return "", err
	}

	c.refreshUTXOs()

	unsignedTransactions, err := c.createUnsignedTransactions(address, sendAmountSompi, false,
		[]string{}, false, nil)
	if err != nil {
		return "", err
	}

	mnemonics := []string{"seed sun dice artwork mango length sudden trial shove wolf dove during aerobic embark copy border unveil convince cost civil there wrong echo front"}

	signedTransactions := make([][]byte, len(unsignedTransactions))
	for i, unsignedTransaction := range unsignedTransactions {
		fmt.Println(hex.EncodeToString(unsignedTransaction))

		signedTransaction, err := libkaspawallet.Sign(c.params, mnemonics, unsignedTransaction, false)
		if err != nil {
			return "", err
		}
		fmt.Println(hex.EncodeToString(signedTransaction))

		signedTransactions[i] = signedTransaction
	}

	fmt.Printf("Broadcasting %d transaction(s)\n", len(signedTransactions))

	const chunkSize = 100 // To avoid sending a message bigger than the gRPC max message size, we split it to chunks
	for offset := 0; offset < len(signedTransactions); offset += chunkSize {
		end := len(signedTransactions)
		if offset+chunkSize <= len(signedTransactions) {
			end = offset + chunkSize
		}

		chunk := signedTransactions[offset:end]
		fmt.Println(hex.EncodeToString(chunk[0]))
		txIDs, err := c.broadcast(chunk, false)
		if err != nil {
			return "", err
		}

		fmt.Printf("Broadcasted %d transaction(s) (broadcasted %.2f%% of the transactions so far)\n", len(chunk), 100*float64(end)/float64(len(signedTransactions)))
		fmt.Println("Broadcasted Transaction ID(s): ")
		for _, txID := range txIDs {
			fmt.Printf("\t%s\n", txID)
			return txID, nil
		}
	}

	return "", fmt.Errorf("not found")

}

func (c *Client) GetBlob(txHash string) ([]byte, error) {
	return nil, nil
}
