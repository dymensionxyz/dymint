package avail

import (
	"errors"
	"fmt"

	"github.com/availproject/avail-go-sdk/metadata"
	prim "github.com/availproject/avail-go-sdk/primitives"
	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/dymensionxyz/dymint/da"
	"github.com/vedhavyas/go-subkey/v2"
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
// It loads the seed/mnemonic from either mnemonic_env or keypath_env.
func NewClient(config *Config) (AvailClient, error) {
	sdk, err := availgo.NewSDK(config.RpcEndpoint)
	if err != nil {
		return nil, err
	}

	seed, err := loadSeed(config)
	if err != nil {
		return nil, fmt.Errorf("load seed: %w", err)
	}

	acc, err := availgo.Account.NewKeyPair(seed)
	if err != nil {
		return nil, err
	}

	client := Client{
		sdk:     sdk,
		account: acc,
		appId:   config.AppID,
	}
	return client, nil
}

// loadSeed loads the seed/mnemonic from the configured source.
func loadSeed(config *Config) (string, error) {
	if err := config.KeyConfig.Validate(); err != nil {
		return "", err
	}

	// Try mnemonic (direct or from file)
	mnemonic, err := config.KeyConfig.GetMnemonic()
	if err != nil {
		return "", err
	}
	if mnemonic != "" {
		return mnemonic, nil
	}

	return "", fmt.Errorf("mnemonic is required for Avail DA: set mnemonic or mnemonic_path")
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
