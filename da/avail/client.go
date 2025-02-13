package avail

import (
	"errors"

	"github.com/availproject/avail-go-sdk/metadata"
	prim "github.com/availproject/avail-go-sdk/primitives"
	"github.com/availproject/avail-go-sdk/sdk"
	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/dymensionxyz/dymint/da"
	"github.com/vedhavyas/go-subkey/v2"
)

type AvailClient interface {
	SubmitData(data []byte) (string, error)
	GetFinalizedHead() (prim.H256, error)
	GetBlock(blockHash string) (sdk.Block, error)
	GetAccountAddress() string
	GetBlobsBySigner(blockHash string, accountAddress string) ([]availgo.DataSubmission, error)
}

type Client struct {
	sdk     sdk.SDK
	account subkey.KeyPair
	appId   uint32
}

var _ AvailClient = &Client{}

// NewClient returns a DA avail client
func NewClient(endpoint string, seed string, appId uint32) (AvailClient, error) {
	sdk, err := sdk.NewSDK(endpoint)
	if err != nil {
		return nil, err
	}
	acc, err := availgo.Account.NewKeyPair(seed)
	if err != nil {
		return nil, err
	}

	client := Client{
		sdk:     sdk,
		account: acc,
		appId:   appId,
	}
	return client, nil
}

func (c Client) SubmitData(data []byte) (string, error) {
	tx := c.sdk.Tx.DataAvailability.SubmitData(data)
	res, err := tx.ExecuteAndWatchInclusion(c.account, availgo.NewTransactionOptions().WithAppId(c.appId))
	if err != nil {
		return "", err
	}
	return res.BlockHash.String(), nil
}

func (c Client) GetFinalizedHead() (prim.H256, error) {
	return c.sdk.Client.Rpc.Chain.GetFinalizedHead()
}

func (c Client) GetBlock(blockHash string) (sdk.Block, error) {
	hash, err := prim.NewH256FromHexString(blockHash)
	if err != nil {
		return sdk.Block{}, errors.Join(da.ErrProofNotMatching, err)
	}

	block, err := sdk.NewBlock(c.sdk.Client, hash)
	if err != nil {
		return sdk.Block{}, errors.Join(da.ErrRetrieval, err)
	}
	return block, nil
}

func (c Client) GetAccountAddress() string {
	return c.account.SS58Address(42)
}

func (c Client) GetBlobsBySigner(blockHash string, accountAddress string) ([]availgo.DataSubmission, error) {
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
