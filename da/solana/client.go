package solana

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	confirm "github.com/gagliardetto/solana-go/rpc/sendAndConfirmTransaction"
	"github.com/gagliardetto/solana-go/rpc/ws"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type SolanaClient interface {
	SubmitBlob(blob []byte) (string, string, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
	GetSignerBalance() (*big.Int, error)
	GetBalance() uint64
	// ValidateInclusion(txHash string, commitment []byte, proof []byte) error
}

var _ SolanaClient = &Client{}

type Client struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
	ctx       context.Context
	cfg       *Config
	pkey      *solana.PrivateKey
}

func NewClient(ctx context.Context, config *Config) (SolanaClient, error) {

	endpoint := rpc.DevNet_RPC
	rpcClient := rpc.New(endpoint)

	keyPath := os.Getenv(config.KeyPathEnv)
	if keyPath == "" {
		return nil, fmt.Errorf("keyPath environment variable %s is not set or empty", config.KeyPathEnv)
	}

	// Load sender's private key (from file, base58, or other means)
	sender, err := solana.PrivateKeyFromSolanaKeygenFile(keyPath)
	if err != nil {
		log.Fatalf("Failed to load keypair: %v", err)
	}

	wsClient, err := ws.Connect(context.Background(), rpc.DevNet_WS)
	if err != nil {
		panic(err)
	}

	client := &Client{
		rpcClient: rpcClient,
		cfg:       config,
		wsClient:  wsClient,
		pkey:      &sender,
	}

	return client, nil
}

func (c *Client) SubmitBlob(blob []byte) (string, string, error) {

	programID := solana.MustPublicKeyFromBase58("J9sagyiVwBwGB1DoaD6T38W6CWPbLfgBafExRnqSKDE")
	// Create instruction with custom data
	data := []byte("HelloGagliardetto")
	instruction := solana.NewInstruction(
		programID,
		[]*solana.AccountMeta{
			{PublicKey: c.pkey.PublicKey(), IsSigner: true, IsWritable: true},
		},
		data,
	)

	// Build transaction
	recentBlockhash, err := c.rpcClient.GetRecentBlockhash(context.Background(), rpc.CommitmentFinalized)

	if err != nil {
		return "", "", err
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(c.pkey.PublicKey()),
	)

	if err != nil {
		return "", "", err
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
		panic(fmt.Errorf("unable to sign transaction: %w", err))
	}

	sig, err := confirm.SendAndConfirmTransaction(
		c.ctx,
		c.rpcClient,
		c.wsClient,
		tx,
	)
	return sig.String(), "", nil
}

func (c *Client) GetBlob(txHash string) ([]byte, error) {

	txSig := solana.MustSignatureFromBase58(txHash)

	out, err := c.rpcClient.GetTransaction(
		c.ctx,
		txSig,
		&rpc.GetTransactionOpts{
			Commitment: rpc.CommitmentConfirmed,
		},
	)
	if err != nil {
		return nil, err
	}

	// Check if logs are present
	if out == nil || out.Meta == nil || len(out.Meta.LogMessages) == 0 {
		return nil, fmt.Errorf("No logs found for this transaction.")

	}
	return []byte(out.Meta.LogMessages[0]), nil

}

func (c *Client) GetSignerBalance() (*big.Int, error) {
	return big.NewInt(0), nil
}

func (c *Client) GetAccountAddress() string {
	return ""
}

func (c *Client) GetBalance() uint64 {
	return uint64(0)
}
