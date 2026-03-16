package solana

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/decred/base58"
	"github.com/dymensionxyz/dymint/da"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"golang.org/x/time/rate"
)

const maxTxData = 973 // 1232 max tx size - 195 bytes (64 signature + 3 header + 96 accounts + 32 blockhash + 64 tx string)

type SolanaClient interface {
	SubmitBlob(blob []byte) (string, string, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
	GetBalance() (uint64, error)
}

var _ SolanaClient = &Client{}

type Client struct {
	submitTxRpcClient  *rpc.Client
	requestTxRpcClient *rpc.Client
	ctx                context.Context
	cfg                *Config
	pkey               *solana.PrivateKey
	programId          *solana.PublicKey
}

// struct used for testability
type RPCClient struct {
	*rpc.Client
	Origin string
}

// NewClient creates the new client that is used to communicate with Solana chain
func NewClient(ctx context.Context, config *Config) (SolanaClient, error) {
	// create two rpc clients, one for sending transactions and one for queries. done this way to allow different rate limits.
	txRpcClient := setRpcClient(config.Endpoint, config.ApiKey, config.SubmitTxRatePerSecond)
	reqRpcClient := setRpcClient(config.Endpoint, config.ApiKey, config.RequestTxRatePerSecond)

	sender, err := loadPrivateKey(config)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	// programId is created from config address
	programID := solana.MustPublicKeyFromBase58(config.ProgramAddress)

	client := &Client{
		ctx:                ctx,
		submitTxRpcClient:  txRpcClient.Client,
		requestTxRpcClient: reqRpcClient.Client,
		cfg:                config,
		pkey:               &sender,
		programId:          &programID,
	}

	return client, nil
}

// loadPrivateKey loads the Solana private key from the configured private key file.
// Solana DA only supports private key file (JSON format with "private_key" field containing base58-encoded key).
func loadPrivateKey(config *Config) (solana.PrivateKey, error) {
	if err := config.KeyConfig.Validate(); err != nil {
		return solana.PrivateKey{}, err
	}

	privateKey, err := config.KeyConfig.GetPrivateKey()
	if err != nil {
		return solana.PrivateKey{}, err
	}
	if privateKey == "" {
		return solana.PrivateKey{}, fmt.Errorf("keypath is required for Solana DA (mnemonic not supported)")
	}

	// Parse the base58-encoded private key
	sender, err := solana.PrivateKeyFromBase58(privateKey)
	if err != nil {
		return solana.PrivateKey{}, fmt.Errorf("parse private key: %w", err)
	}
	return sender, nil
}

// SubmitBlob slices the blob in small pieces and sends each piece to the Solana program (specified in config) as input data. It returns the list of transactions plus the blob hash used to compare with the original one on retrieval.
func (c *Client) SubmitBlob(blob []byte) (string, string, error) {
	// generate the transactions with blob data and sends them to Solana
	txHash, err := c.generateAndSubmitBlobTxs(blob)
	if err != nil {
		return "", "", err
	}

	// calculates the blob hash
	h := sha256.New()
	h.Write(blob)
	blobHash := h.Sum(nil)
	blobHashString := hex.EncodeToString(blobHash)

	return txHash, blobHashString, nil
}

// GetBlob gets the input data from each transaction included in the txHash, from the Solana transaction logs, and aggregates them to regenerate the blob.
func (c *Client) GetBlob(txHash string) ([]byte, error) {
	var hexResult strings.Builder
	var data []string
	for {
		result, tx, err := c.getDataFromTxLogs(txHash)
		if err != nil {
			return nil, err
		}
		data = append(data, result)
		if tx == "" {
			break
		}
		txHash = tx
	}
	for i := len(data) - 1; i >= 0; i-- {
		hexResult.WriteString(data[i])
	}

	blob, err := hex.DecodeString(hexResult.String())
	if err != nil {
		return nil, errors.Join(da.ErrBlobNotParsed, fmt.Errorf("unable to decode hex payload"))
	}
	return blob, nil
}

// GetAccountAddress returns the Solana address derived from the private key
func (c *Client) GetAccountAddress() string {
	return c.pkey.PublicKey().String()
}

// GetBalance returns the address balance in lamports
func (c *Client) GetBalance() (uint64, error) {
	resp, err := c.requestTxRpcClient.GetBalance(
		c.ctx,
		c.pkey.PublicKey(),
		rpc.CommitmentFinalized,
	)
	if err != nil {
		return uint64(0), err
	}
	// Balance is in lamports (1 SOL = 1_000_000_000 lamports)
	return resp.Value, nil
}

// generateAndSubmitBlobTxs splits the blob, based on maximum data that can be included in a transaction (maxTxData), and creates and sends every piece in a single transaction.
func (c *Client) generateAndSubmitBlobTxs(blob []byte) (string, error) {
	blobHex := []byte(hex.EncodeToString(blob))

	// this calculates the number of txs necessary and creates them (based on available size for the payload), chunking the blob and including a part of it into each tx sequentially
	splitCount := len(blobHex) / maxTxData

	// adds another if the split is not exact
	if len(blobHex)%maxTxData > 0 {
		splitCount++
	}
	var txHashes []string

	for i := range splitCount {

		// calculate start end byte for the payload
		startChunkIndex := i * maxTxData
		endChunkIndex := startChunkIndex + maxTxData
		if endChunkIndex > len(blobHex) {
			endChunkIndex = len(blobHex)
		}
		data := blobHex[startChunkIndex:endChunkIndex]

		// gets the recent block hash that needs to be included in the transaction
		recentBlockhash, err := c.requestTxRpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
		if err != nil {
			return "", err
		}
		var txSigBytes []byte
		if i > 0 {
			txSigBytes = []byte(txHashes[i-1])
		} else {
			txSigBytes = []byte(base58.Encode([]byte{}))
		}
		payload := append([]byte{byte(len(txSigBytes))}, txSigBytes...)
		payload = append(payload, data...)

		// it creates the instruction with the payload (blob chunk) as input data
		instruction := solana.NewInstruction(
			*c.programId,
			[]*solana.AccountMeta{},
			payload,
		)

		// it creates the transaction with the previously created instruction (in Solana one transaction can include multiple instructions)
		tx, err := solana.NewTransaction(
			[]solana.Instruction{instruction},
			recentBlockhash.Value.Blockhash,
			solana.TransactionPayer(c.pkey.PublicKey()),
		)
		if err != nil {
			return "", err
		}

		// sign the transaction
		_, err = tx.Sign(
			func(key solana.PublicKey) *solana.PrivateKey {
				if c.pkey.PublicKey().Equals(key) {
					return c.pkey
				}
				return nil
			},
		)
		if err != nil {
			return "", fmt.Errorf("unable to sign transaction. err: %w", err)
		}

		// sends the transaction to rpc node (without waiting for confirmation to make it faster -rate will be controlled by rate limiter if used-). Confirmation will be validated in CheckAvailability().
		// TODO: add enhanced mechanism to avoid retransmitting the whole blob in case of failures https://github.com/dymensionxyz/dymint/issues/1436
		sig, err := c.submitTxRpcClient.SendTransaction(c.ctx, tx)
		if err != nil {
			return "", fmt.Errorf("unable to send and confirm transaction. err: %w", err)
		}
		txHashes = append(txHashes, sig.String())
	}

	return txHashes[len(txHashes)-1], nil
}

func (c *Client) getDataFromTxLogs(txHash string) (string, string, error) {
	txSig := solana.MustSignatureFromBase58(txHash)

	out, err := c.requestTxRpcClient.GetTransaction(
		c.ctx,
		txSig,
		&rpc.GetTransactionOpts{
			Commitment: rpc.CommitmentConfirmed,
		},
	)
	if errors.Is(err, fmt.Errorf("not found")) {
		return "", "", da.ErrBlobNotFound
	}

	if err != nil {
		return "", "", err
	}

	// Check if logs are present
	if out == nil || out.Meta == nil || len(out.Meta.LogMessages) == 0 {
		return "", "", errors.Join(da.ErrBlobNotFound, fmt.Errorf("no logs found for this transaction"))
	}

	hash, found := strings.CutPrefix(out.Meta.LogMessages[1], "Program log: Tx: ")
	if !found {
		return "", "", errors.Join(da.ErrBlobNotFound, fmt.Errorf("next tx not found in transaction logs"))
	}

	data, found := strings.CutPrefix(out.Meta.LogMessages[2], "Program log: Data: ")
	if !found {
		return "", "", errors.Join(da.ErrBlobNotFound, fmt.Errorf("data not found in transaction logs"))
	}

	return data, hash, nil
}

func setRpcClient(endpoint string, apiKey string, maxRatePerSecond *int) *RPCClient {
	if apiKey != "" && maxRatePerSecond != nil {
		jsonRpcClient := rpc.NewWithLimiterWithCustomHeaders(endpoint, rate.Every(time.Second), *maxRatePerSecond, map[string]string{
			"x-api-key": apiKey,
		})
		return &RPCClient{rpc.NewWithCustomRPCClient(jsonRpcClient), "limiter+apikey"}
	}

	if apiKey != "" {
		return &RPCClient{rpc.NewWithHeaders(endpoint, map[string]string{
			"x-api-key": apiKey,
		}), "apikey"}
	}

	if maxRatePerSecond != nil {
		jsonRpcClient := rpc.NewWithLimiter(endpoint, rate.Every(time.Second), *maxRatePerSecond)
		return &RPCClient{rpc.NewWithCustomRPCClient(jsonRpcClient), "limiter"}
	}

	return &RPCClient{rpc.New(endpoint), "default"}
}
