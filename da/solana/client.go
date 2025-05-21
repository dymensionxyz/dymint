package solana

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"golang.org/x/time/rate"

	"github.com/gagliardetto/solana-go/rpc"
)

const maxTxData = 1037 // 1232 max tx size - 195 bytes (64 signature + 3 header + 96 accounts + 32 blockhash)

type SolanaClient interface {
	SubmitBlob(blob []byte) ([]string, string, error)
	GetBlob(txHash []string) ([]byte, error)
	GetAccountAddress() string
	GetSignerBalance() (*big.Int, error)
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

func NewClient(ctx context.Context, config *Config) (SolanaClient, error) {
	var jsonTxRpcClient rpc.JSONRPCClient
	var jsonReqRpcClient rpc.JSONRPCClient

	if os.Getenv(config.ApiKeyEnv) != "" {
		apiKey := os.Getenv(config.ApiKeyEnv)
		jsonTxRpcClient = rpc.NewWithLimiterWithCustomHeaders(config.Endpoint, rate.Every(time.Second), *config.SubmitTxRate, map[string]string{
			"x-api-key": apiKey,
		})
		jsonReqRpcClient = rpc.NewWithLimiterWithCustomHeaders(config.Endpoint, rate.Every(time.Second), *config.RequestTxRate, map[string]string{
			"x-api-key": apiKey,
		})
	} else {
		jsonTxRpcClient = rpc.NewWithLimiter(config.Endpoint, rate.Every(time.Second), *config.SubmitTxRate)
		jsonReqRpcClient = rpc.NewWithLimiter(config.Endpoint, rate.Every(time.Second), *config.RequestTxRate)
	}

	txRpcClient := rpc.NewWithCustomRPCClient(jsonTxRpcClient)
	reqRpcClient := rpc.NewWithCustomRPCClient(jsonReqRpcClient)

	keyPath := os.Getenv(config.KeyPathEnv)
	if keyPath == "" {
		return nil, fmt.Errorf("keyPath environment variable %s is not set or empty", config.KeyPathEnv)
	}

	// Load sender's private key (from file, base58, or other means)
	sender, err := solana.PrivateKeyFromSolanaKeygenFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to load keypair: %v", err)
	}

	programID := solana.MustPublicKeyFromBase58(config.ProgramAddress)

	client := &Client{
		ctx:                ctx,
		submitTxRpcClient:  txRpcClient,
		requestTxRpcClient: reqRpcClient,
		cfg:                config,
		pkey:               &sender,
		programId:          &programID,
	}

	return client, nil
}

func (c *Client) SubmitBlob(blob []byte) ([]string, string, error) {
	txHash, err := c.generateAndSubmitBlobTxs(blob)
	if err != nil {
		return nil, "", err
	}

	h := sha256.New()
	h.Write(blob)
	blobHash := h.Sum(nil)
	blobHashString := hex.EncodeToString(blobHash)

	return txHash, blobHashString, nil
}

func (c *Client) GetBlob(txHash []string) ([]byte, error) {
	var hexResult strings.Builder
	for _, hash := range txHash {
		result, err := c.getDataFromTxLogs(hash)
		if err != nil {
			return nil, err
		}
		hexResult.WriteString(result)
	}
	blob, err := hex.DecodeString(hexResult.String())
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (c *Client) GetSignerBalance() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (c *Client) GetAccountAddress() string {
	return ""
}

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

func (c *Client) generateAndSubmitBlobTxs(blob []byte) ([]string, error) {
	blobHex := []byte(hex.EncodeToString(blob))

	// this calculates the number of txs necessary and creates them (based on available size for the payload), chunking the blob and including a part of it into each tx sequentially
	splitCount := len(blobHex) / maxTxData

	if len(blobHex)%maxTxData > 0 {
		splitCount++
	}
	var txHash []string
	for i := range splitCount {

		startChunkIndex := i * maxTxData
		endChunkIndex := startChunkIndex + maxTxData
		if endChunkIndex > len(blobHex) {
			endChunkIndex = len(blobHex)
		}
		payload := blobHex[startChunkIndex:endChunkIndex]

		// Build transaction
		recentBlockhash, err := c.requestTxRpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentFinalized)
		if err != nil {
			return nil, err
		}

		instruction := solana.NewInstruction(
			*c.programId,
			[]*solana.AccountMeta{},
			payload,
		)

		tx, err := solana.NewTransaction(
			[]solana.Instruction{instruction},
			recentBlockhash.Value.Blockhash,
			solana.TransactionPayer(c.pkey.PublicKey()),
		)
		if err != nil {
			return nil, err
		}

		_, err = tx.Sign(
			func(key solana.PublicKey) *solana.PrivateKey {
				if c.pkey.PublicKey().Equals(key) {
					return c.pkey
				}
				return nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("unable to sign transaction. err: %w", err)
		}

		sig, err := c.submitTxRpcClient.SendTransaction(c.ctx, tx)
		if err != nil {
			return nil, fmt.Errorf("unable to send and confirm transaction. err: %w", err)
		}
		txHash = append(txHash, sig.String())
	}

	return txHash, nil
}

func (c *Client) getDataFromTxLogs(txHash string) (string, error) {
	txSig := solana.MustSignatureFromBase58(txHash)

	out, err := c.requestTxRpcClient.GetTransaction(
		c.ctx,
		txSig,
		&rpc.GetTransactionOpts{
			Commitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return "", err
	}

	// Check if logs are present
	if out == nil || out.Meta == nil || len(out.Meta.LogMessages) == 0 {
		return "", fmt.Errorf("No logs found for this transaction.")
	}
	result, found := strings.CutPrefix(out.Meta.LogMessages[1], "Program log: ")
	if !found {
		return "", fmt.Errorf("unable to cut program log string")
	}

	return result, nil
}
