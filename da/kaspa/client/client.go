package client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/bip32"
	"github.com/kaspanet/kaspad/domain/consensus/model/externalapi"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/kaspanet/kaspad/util/txmass"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
)

const (
	minChangeTarget = constants.SompiPerKaspa * 10
	minFeeRate      = 1.0
	//address                             = "kaspatest:qp75u7cuphjwyq9j6ghe2v0j3gtvxlppyurq279h4ckpdc7umdh6vrusw9c7d"
	numIndexesToQueryForFarAddresses    = 100
	numIndexesToQueryForRecentAddresses = 1000
	// Purpose and CoinType constants
	SingleSignerPurpose = 44
	// Note: this is not entirely compatible to BIP 45 since
	// TODO: Register the coin type in https://github.com/satoshilabs/slips/blob/master/slip-0044.md
	CoinType = 111111

	TRANSIENT_BYTE_TO_MASS_FACTOR = 4
	fromAddress                   = "kaspatest:qp75u7cuphjwyq9j6ghe2v0j3gtvxlppyurq279h4ckpdc7umdh6vrusw9c7d"
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
	rpcClient        *rpcclient.RPCClient // RPC client for ongoing user requests
	httpClient       *http.Client
	params           *dagconfig.Params
	coinbaseMaturity uint64 // Is different from default if we use testnet-11
	usedOutpoints    map[externalapi.DomainOutpoint]time.Time
	//startTimeOfLastCompletedRefresh time.Time
	apiURL              string
	utxosSortedByAmount []*walletUTXO
	extendedKey         *bip32.ExtendedKey
	//keysFile                        *keys.File
	txMassCalculator     *txmass.Calculator
	mempoolExcludedUTXOs map[externalapi.DomainOutpoint]*walletUTXO
	addressSet           walletAddressSet
	//nextSyncStartIndex   uint32
}

var _ KaspaClient = &Client{}

func NewClient(ctx context.Context, config *Config, mnemonic string) (KaspaClient, error) {
	rpcClient, err := rpcclient.NewRPCClient(config.GrpcAddress)
	if err != nil {
		return nil, err
	}

	if config.Timeout != 0 {
		rpcClient.SetTimeout(time.Duration(config.Timeout) * time.Second)
	}

	params := &dagconfig.TestnetParams
	seed := bip39.NewSeed(mnemonic, "")
	version, err := versionFromParams(params)
	if err != nil {
		return nil, err
	}

	master, err := bip32.NewMasterWithPath(seed, version, defaultPath())
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	kaspaClient := &Client{
		rpcClient:        rpcClient,
		httpClient:       httpClient,
		coinbaseMaturity: 100,
		extendedKey:      master,
		params:           params,
		apiURL:           config.RPCURL,
		addressSet:       make(walletAddressSet),
		txMassCalculator: txmass.NewCalculator(1, 10, 1000),
	}
	return kaspaClient, nil

}

func (c *Client) Stop() {
	c.rpcClient.Disconnect()
}

func (c *Client) SubmitBlob(blob []byte) (string, error) {

	c.refreshUTXOs()

	unsignedTransactions, err := c.createUnsignedTransactions(fromAddress, blob)
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

	/*const chunkSize = 100 // To avoid sending a message bigger than the gRPC max message size, we split it to chunks
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
	}*/
	txIds, err := c.broadcast(signedTransactions, false)
	if err != nil {
		return "", err
	}
	return txIds[0], nil
	//return "", fmt.Errorf("not found")

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

func versionFromParams(params *dagconfig.Params) ([4]byte, error) {
	switch params.Name {
	case dagconfig.MainnetParams.Name:
		return bip32.KaspaMainnetPrivate, nil
	case dagconfig.TestnetParams.Name:
		return bip32.KaspaTestnetPrivate, nil
	case dagconfig.DevnetParams.Name:
		return bip32.KaspaDevnetPrivate, nil
	case dagconfig.SimnetParams.Name:
		return bip32.KaspaSimnetPrivate, nil
	}

	return [4]byte{}, errors.Errorf("unknown network %s", params.Name)
}

func defaultPath() string {
	return fmt.Sprintf("m/%d'/%d'/0'", SingleSignerPurpose, CoinType)
}
