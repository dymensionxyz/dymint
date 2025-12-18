package avail

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/availproject/avail-go-sdk/metadata"
	prim "github.com/availproject/avail-go-sdk/primitives"
	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/dymensionxyz/dymint/da"
	"github.com/vedhavyas/go-subkey/v2"
	"github.com/vedhavyas/go-subkey/v2/sr25519"
)

type AvailClient interface {
	SubmitData(data []byte) (string, error)
	IsSyncing() (bool, error)
	GetBlock(blockHash string) (availgo.Block, error)
	GetAccountAddress() string
	GetBlobsBySigner(blockHash string, accountAddress string) ([]availgo.DataSubmission, error)
}

type Client struct {
	sdk     availgo.SDK
	account subkey.KeyPair
	appId   uint32
}

var _ AvailClient = &Client{}

// NewClient returns a DA avail client using the provided config.
// Supports both mnemonic (direct or from file) and private key (from JSON file).
func NewClient(config *Config) (AvailClient, error) {
	sdk, err := availgo.NewSDK(config.RpcEndpoint)
	if err != nil {
		return nil, err
	}

	acc, err := loadKeyPair(config)
	if err != nil {
		return nil, fmt.Errorf("load keypair: %w", err)
	}

	client := Client{
		sdk:     sdk,
		account: acc,
		appId:   config.AppID,
	}
	return client, nil
}

// loadKeyPair loads the keypair from the configured source.
// Supports mnemonic (direct or from file) and private key (from JSON file).
func loadKeyPair(config *Config) (subkey.KeyPair, error) {
	if err := config.KeyConfig.Validate(); err != nil {
		return nil, err
	}

	// Try mnemonic first (direct or from file)
	mnemonic, err := config.KeyConfig.GetMnemonic()
	if err != nil {
		return nil, err
	}
	if mnemonic != "" {
		return availgo.Account.NewKeyPair(mnemonic)
	}

	// Try private key from JSON file
	privateKey, err := config.KeyConfig.GetPrivateKey()
	if err != nil {
		return nil, err
	}
	if privateKey != "" {
		return keyPairFromHex(privateKey)
	}

	return nil, fmt.Errorf("key configuration required for Avail DA: set mnemonic, mnemonic_path, or keypath")
}

// keyPairFromHex creates a sr25519 keypair from a hex-encoded private key/seed.
func keyPairFromHex(hexKey string) (subkey.KeyPair, error) {
	// Remove 0x prefix if present
	hexKey = strings.TrimPrefix(hexKey, "0x")

	seed, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode hex key: %w", err)
	}

	scheme := sr25519.Scheme{}
	return scheme.FromSeed(seed)
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitData(data []byte) (string, error) {
	syncing, err := c.IsSyncing()
	if syncing || err != nil {
		return "", errors.Join(err, da.ErrDANotAvailable)
	}
	tx := c.sdk.Tx.DataAvailability.SubmitData(data)
	res, err := tx.ExecuteAndWatchInclusion(c.account, availgo.NewTransactionOptions().WithAppId(c.appId))
	if err != nil {
		return "", err
	}
	return res.BlockHash.String(), nil
}

// GetBlock retrieves a block from Avail chain by block hash
func (c Client) GetBlock(blockHash string) (availgo.Block, error) {
	hash, err := prim.NewH256FromHexString(blockHash)
	if err != nil {
		return availgo.Block{}, errors.Join(da.ErrProofNotMatching, err)
	}

	block, err := availgo.NewBlock(c.sdk.Client, hash)
	if err != nil {
		return availgo.Block{}, errors.Join(da.ErrRetrieval, err)
	}
	return block, nil
}

// IsSyncing returns true if remote rpc node is still syncing
func (c Client) IsSyncing() (bool, error) {
	value, err := c.sdk.Client.Rpc.System.Health()
	if err != nil {
		return false, err
	}
	return value.IsSyncing, nil
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return c.account.SS58Address(42)
}

// GetBlobsBySigner returns posted blobs filtered by block and sequencer account
func (c Client) GetBlobsBySigner(blockHash string, accountAddress string) ([]availgo.DataSubmission, error) {
	syncing, err := c.IsSyncing()
	if syncing || err != nil {
		return nil, errors.Join(err, da.ErrDANotAvailable)
	}

	block, err := c.GetBlock(blockHash)
	if err != nil {
		return nil, err
	}
	accountId, err := metadata.NewAccountIdFromAddress(accountAddress)
	if err != nil {
		return nil, err
	}

	// Block Blobs filtered by Signer
	return block.DataSubmissionBySigner(accountId), nil
}
