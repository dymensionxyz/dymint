package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dymensionxyz/dymint/da"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet/bip32"
	"github.com/kaspanet/kaspad/domain/consensus/utils/constants"
	"github.com/kaspanet/kaspad/domain/dagconfig"
	"github.com/kaspanet/kaspad/infrastructure/network/rpcclient"
	"github.com/kaspanet/kaspad/util"
	"github.com/kaspanet/kaspad/util/txmass"
	"github.com/tyler-smith/go-bip39"
)

const (
	minChangeTarget               = constants.SompiPerKaspa * 10
	SingleSignerPurpose           = 44
	CoinType                      = 111111
	TRANSIENT_BYTE_TO_MASS_FACTOR = 4
	Mainnet                       = "kaspa-mainnet"
	Testnet                       = "kaspa-testnet-10"
	TxHashLength                  = 64
	ConfirmationsRequired         = 1000 // Kaspa requires 1000 confirmations (after Crescent hardfork) to consider a transaction final and safe against reorgs.
	KaspaBlocksPerSecond          = 10
)

type KaspaClient interface {
	Stop() error
	GetBalance() (uint64, error)
	SubmitBlob(blob []byte) ([]string, string, error)
	GetBlob(txHash []string) ([]byte, error)
	CheckTransactionMaturity(txHash []string) error
}

// Transaction is a partial struct to extract payload and maturity info
type Transaction struct {
	TransactionID           string `json:"transaction_id"`
	Payload                 string `json:"payload"`
	BlockTime               uint64 `json:"block_time,omitempty"`
	AcceptingBlockHash      string `json:"accepting_block_hash,omitempty"`
	IsCoinbase              bool   `json:"is_coinbase,omitempty"`
	AcceptingBlockBlueScore uint64 `json:"accepting_block_blue_score,omitempty"`
	IsAccepted              bool   `json:"is_accepted,omitempty"`
}

type FailedTxRetrieve struct {
	Result string `json:"detail"`
}

type Client struct {
	rpcClient        *rpcclient.RPCClient // RPC client for ongoing user requests
	httpClient       *http.Client
	params           *dagconfig.Params
	apiURL           string
	address          util.Address
	wAddress         *walletAddress
	mnemonic         string
	publicKey        *bip32.ExtendedKey
	txMassCalculator *txmass.Calculator
}

var _ KaspaClient = &Client{}

func NewClient(ctx context.Context, config *Config, mnemonic string) (KaspaClient, error) {
	rpcClient, err := rpcclient.NewRPCClient(config.GrpcAddress)
	if err != nil {
		return nil, err
	}

	if config.Timeout != 0 {
		rpcClient.SetTimeout(config.Timeout)
	}

	var params *dagconfig.Params
	switch config.Network {
	case Testnet:
		params = &dagconfig.TestnetParams
	case Mainnet:
		params = &dagconfig.MainnetParams
	default:
		return nil, fmt.Errorf("kaspa network not set to testnet or mainnet. Param: %s", config.Network)
	}

	seed := bip39.NewSeed(mnemonic, "")
	version, err := versionFromNetworkName(params.Name)
	if err != nil {
		return nil, err
	}

	master, err := bip32.NewMasterWithPath(seed, version, defaultPath())
	if err != nil {
		return nil, err
	}

	address, err := util.DecodeAddress(config.Address, params.Prefix)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	pubKey, err := master.Public()
	if err != nil {
		return nil, err
	}

	kaspaClient := &Client{
		rpcClient:        rpcClient,
		httpClient:       httpClient,
		publicKey:        pubKey,
		address:          address,
		params:           params,
		mnemonic:         mnemonic,
		apiURL:           config.APIUrl,
		txMassCalculator: txmass.NewCalculator(1, 10, 1000),
	}
	return kaspaClient, nil
}

// Stop disconnects from grpc
func (c *Client) Stop() error {
	return c.rpcClient.Disconnect()
}

// SubmitBlob sends the blob to Kaspa network, including the blob in  Kaspa Txs
func (c *Client) SubmitBlob(blob []byte) ([]string, string, error) {
	// generate txs
	blobTxs, err := c.generateBlobTxs(blob)
	if err != nil {
		return nil, "", err
	}

	// sign tx
	signedTxs := make([][]byte, len(blobTxs))
	for i, tx := range blobTxs {
		signedTx, err := libkaspawallet.Sign(c.params, []string{c.mnemonic}, tx, false)
		if err != nil {
			return nil, "", err
		}
		signedTxs[i] = signedTx
	}

	// send txs to Kaspa node
	txIds, err := c.broadcast(signedTxs)
	if err != nil {
		return nil, "", err
	}

	h := sha256.New()
	h.Write(blob)
	blobHash := h.Sum(nil)
	// return tx ids obtained
	return txIds, hex.EncodeToString(blobHash), nil
}

// GetBlob retrieves the blob from Kaspa network, by tx ids
func (c *Client) GetBlob(txHash []string) ([]byte, error) {
	txData := make([][]byte, len(txHash))
	for i, hash := range txHash {
		tx, err := c.retrieveBlobTx(hash)
		if err != nil {
			return nil, err
		}
		txData[i], err = hex.DecodeString(tx.Payload)
		if err != nil {
			return nil, err
		}
	}
	var blob []byte
	for _, data := range txData {
		blob = append(blob, data...)
	}
	return blob, nil
}

func (c *Client) GetBalance() (uint64, error) {
	balance := uint64(0)
	utxos, err := c.getUTXOs()
	if err != nil {
		return balance, err
	}
	dagInfo, err := c.rpcClient.GetBlockDAGInfo()
	if err != nil {
		return balance, err
	}
	for _, utxo := range utxos {
		if !c.isUTXOSpendable(utxo, dagInfo.VirtualDAAScore) {
			continue
		}

		balance += utxo.UTXOEntry.Amount()
	}
	return balance, nil
}

// APIError represents an error from the Kaspa API with status code
type APIError struct {
	StatusCode int
	Message    string
}

func (e APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// makeAPICall makes a generic API call to the Kaspa REST API
func (c *Client) makeAPICall(endpoint string, result interface{}) error {
	url := fmt.Sprintf("%s%s", c.apiURL, endpoint)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != 200 {
		return APIError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("endpoint %s returned status %d", endpoint, resp.StatusCode)}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("kaspa API response decode failed: %w", err)
	}
	return nil
}

// retrieveBlobTx gets Tx, that includes blob parts, using  Kaspa REST-API server (https://api.kaspa.org/docs)
func (c *Client) retrieveBlobTx(txHash string) (*Transaction, error) {
	if len(txHash) != TxHashLength {
		return nil, da.ErrBlobNotFound
	}

	var tx Transaction
	endpoint := fmt.Sprintf("/transactions/%s", txHash)
	if err := c.makeAPICall(endpoint, &tx); err != nil {
		// Check if this is a 404 error which might mean transaction not found
		var apiErr APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			// Make another call to get the detailed error message
			url := fmt.Sprintf("%s%s", c.apiURL, endpoint)
			resp, _ := c.httpClient.Get(url)
			if resp != nil {
				defer resp.Body.Close() // nolint:errcheck
				var failedTx FailedTxRetrieve
				if json.NewDecoder(resp.Body).Decode(&failedTx) == nil {
					if failedTx.Result == "Transaction not found" {
						return nil, da.ErrBlobNotFound
					}
				}
			}
		}
		return nil, err
	}
	return &tx, nil
}

// returns version params depending on the network used (mainnet or testnet)
func versionFromNetworkName(name string) ([4]byte, error) {
	switch name {
	case Mainnet:
		return bip32.KaspaMainnetPrivate, nil
	case Testnet:
		return bip32.KaspaTestnetPrivate, nil
	}

	return [4]byte{}, fmt.Errorf("kaspa network not valid %s", name)
}

func defaultPath() string {
	return fmt.Sprintf("m/%d'/%d'/0'", SingleSignerPurpose, CoinType)
}

// BlueScoreResponse represents the response from the virtual-chain-blue-score endpoint
type BlueScoreResponse struct {
	BlueScore uint64 `json:"blueScore"`
}

// CheckTransactionMaturity checks if all transactions in the list are mature enough
func (c *Client) CheckTransactionMaturity(txHash []string) error {
	// Get current virtual chain blue score via API
	var blueScoreResp BlueScoreResponse
	if err := c.makeAPICall("/info/virtual-chain-blue-score", &blueScoreResp); err != nil {
		return fmt.Errorf("failed to get virtual chain blue score: %w", err)
	}

	maxMissingConfirmations := uint64(0)

	for _, hash := range txHash {
		tx, err := c.retrieveBlobTx(hash)
		if err != nil {
			return err
		}

		// return early if the transaction is not accepted yet, with missing confirmations set to full required
		if !tx.IsAccepted {
			return da.ErrMaturityNotReached{MissingConfirmations: ConfirmationsRequired}
		}

		// if the transaction's accepting block blue score is higher than current blue score, return maturity not reached, with missing confirmations set to full required
		// it should not happen, but REST API might be lagging for the chain blue score
		if blueScoreResp.BlueScore < tx.AcceptingBlockBlueScore {
			return da.ErrMaturityNotReached{MissingConfirmations: ConfirmationsRequired}
		}

		// if the transaction is mature enough, go to next tx
		if ConfirmationsRequired < blueScoreResp.BlueScore-tx.AcceptingBlockBlueScore {
			continue
		}

		// calculate missing confirmations
		missingConfirmations := ConfirmationsRequired - (blueScoreResp.BlueScore - tx.AcceptingBlockBlueScore)
		// set max missing confirmations from all txs
		if missingConfirmations > maxMissingConfirmations {
			maxMissingConfirmations = missingConfirmations
		}
	}

	if maxMissingConfirmations > 0 {
		return da.ErrMaturityNotReached{MissingConfirmations: maxMissingConfirmations}
	}

	return nil
}
