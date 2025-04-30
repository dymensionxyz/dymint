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
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/kaspanet/kaspad/util/txmass"
	"github.com/pkg/errors"
	"github.com/tyler-smith/go-bip39"
)

const (
	minChangeTarget                     = constants.SompiPerKaspa * 10
	numIndexesToQueryForRecentAddresses = 1000
	SingleSignerPurpose                 = 44
	CoinType                            = 111111
	TRANSIENT_BYTE_TO_MASS_FACTOR       = 4
)

type KaspaClient interface {
	SubmitBlob(blob []byte) (string, error)
	GetBlob(txHash string) ([]byte, error)
	Stop()
	GetBalance() uint64
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
	apiURL           string
	fromAddress      string
	mnemonic         string
	balance          uint64
	extendedKey      *bip32.ExtendedKey
	txMassCalculator *txmass.Calculator
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

	var params *dagconfig.Params
	switch config.Network {
	case "testnet":
		params = &dagconfig.TestnetParams
	case "mainnet":
		params = &dagconfig.MainnetParams
	default:
		return nil, fmt.Errorf("Config network parameter not set to testnet or mainnet. Param: %d", config.Network)
	}

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
		fromAddress:      config.FromAddress,
		mnemonic:         mnemonic,
		params:           params,
		apiURL:           config.APIUrl,
		balance:          uint64(0),
		txMassCalculator: txmass.NewCalculator(1, 10, 1000),
	}
	return kaspaClient, nil

}

func (c *Client) Stop() {
	c.rpcClient.Disconnect()
}

func (c *Client) SubmitBlob(blob []byte) (string, error) {

	utxos, err := c.getUTXOs()
	if err != nil {
		return "", err
	}

	unsignedTransactions, err := c.createUnsignedTransactions(utxos, c.fromAddress, blob)
	if err != nil {
		return "", err
	}

	signedTransactions := make([][]byte, len(unsignedTransactions))
	for i, unsignedTransaction := range unsignedTransactions {
		signedTransaction, err := libkaspawallet.Sign(c.params, []string{c.mnemonic}, unsignedTransaction, false)
		if err != nil {
			return "", err
		}
		signedTransactions[i] = signedTransaction
	}

	txIds, err := c.broadcast(signedTransactions, false)
	if err != nil {
		return "", err
	}
	return txIds[0], nil

}

func (c *Client) GetBlob(txHash string) ([]byte, error) {

	url := fmt.Sprintf(c.apiURL+"/transactions/%s", txHash)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("post failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Http response status code not OK: Status: %d", resp.StatusCode)
	}

	var tx Transaction
	if err := json.NewDecoder(resp.Body).Decode(&tx); err != nil {
		return nil, fmt.Errorf("Kaspa transaction decode failed: %w", err)
	}

	data, err := hex.DecodeString(tx.Payload)
	if err != nil {
		return nil, err
	}
	return data, nil

}

func (c *Client) GetBalance() uint64 {
	return c.balance
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
