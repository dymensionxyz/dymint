package kaspa

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/keys"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/kaspanet/kaspad/util/txmass"
)

const (
	minChangeTarget                     = constants.SompiPerKaspa * 10
	minFeeRate                          = 1.0
	address                             = "kaspatest:qp96y8xa6gqlh3a5c6wu9x73a5egvsw2vk7w7nzm8x98wvkavjlg29zvta4m6"
	numIndexesToQueryForFarAddresses    = 100
	numIndexesToQueryForRecentAddresses = 1000
)

type KaspaClient interface {
	SubmitBlob(blob []byte) (string, error)
	GetBlob(txHash string) ([]byte, error)
	Stop()
}

// Transaction is a partial struct to extract payload
type Transaction struct {
	TransactionID string `json:"transaction_id"`
	Payload       string `json:"payload"`
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
	apiURL                          string
	utxosSortedByAmount             []*walletUTXO
	keysFile                        *keys.File
	txMassCalculator                *txmass.Calculator
	mempoolExcludedUTXOs            map[externalapi.DomainOutpoint]*walletUTXO
	addressSet                      walletAddressSet
	nextSyncStartIndex              uint32
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

	keysFile, err := keys.ReadKeysFile(&dagconfig.Params{}, config.KeysPath)
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
		apiURL:             config.RPCURL,
		addressSet:         make(walletAddressSet),
		txMassCalculator:   txmass.NewCalculator(1, 10, 1000),
		usedOutpoints:      map[externalapi.DomainOutpoint]time.Time{},
	}
	return kaspaClient, nil

}

func (c *Client) Stop() {
	c.rpcClient.Disconnect()
}

func (c *Client) SubmitBlob(blob []byte) (string, error) {

	err := c.collectFarAddresses()
	if err != nil {
		return "", err
	}

	c.refreshUTXOs()

	unsignedTransactions, err := c.createUnsignedTransactions(address, blob)
	if err != nil {
		return "", err
	}

	mnemonics := []string{"seed sun dice artwork mango length sudden trial shove wolf dove during aerobic embark copy border unveil convince cost civil there wrong echo front"}

	signedTransactions := make([][]byte, len(unsignedTransactions))
	for i, unsignedTransaction := range unsignedTransactions {

		signedTransaction, err := libkaspawallet.Sign(c.params, mnemonics, unsignedTransaction, false)
		if err != nil {
			return "", err
		}

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
		txIDs, err := c.broadcast(chunk, false)
		if err != nil {
			return "", err
		}

		for _, txID := range txIDs {
			return txID, nil
		}
	}

	return "", fmt.Errorf("not found")

}

func (c *Client) GetBlob(txHash string) ([]byte, error) {

	url := fmt.Sprintf(c.apiURL+"/%s", txHash)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("post failed: %w", err)
	}
	defer resp.Body.Close()

	var tx Transaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	data, err := hex.DecodeString(tx.Payload)
	if err != nil {
		return nil, err
	}
	return data, nil

}
