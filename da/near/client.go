package avail

import (
	"context"
	"encoding/hex"
	"errors"

	"github.com/availproject/avail-go-sdk/metadata"
	prim "github.com/availproject/avail-go-sdk/primitives"
	"github.com/availproject/avail-go-sdk/sdk"
	availgo "github.com/availproject/avail-go-sdk/sdk"
	"github.com/dymensionxyz/dymint/da"
	"github.com/ethereum/go-ethereum/common"
	near "github.com/near/rollup-data-availability/gopkg/da-rpc"
	"github.com/rs/zerolog/log"
)

type NearClient interface {
	SubmitData(data []byte) (string, error)
	GetBlock(blockHash string) (sdk.Block, error)
	GetAccountAddress() string
	GetBlobsBySigner(blockHash string, accountAddress string) ([]availgo.DataSubmission, error)
}

type NearConfig struct {
	Enable    bool
	Account   string
	Contract  string
	Key       string
	Namespace uint32
}

type Client struct {
	near near.Config
}

var _ NearClient = &Client{}

// NewClient returns a DA avail client
func NewClient(config NearConfig) (NearClient, error) {

	near, err := near.NewConfig(config.Account, config.Contract, config.Key, config.Namespace)
	if err != nil {
		return nil, err
	}

	client := Client{
		near: *near,
	}
	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitData(data []byte) (string, error) {
	frameRefBytes, err := c.near.ForceSubmit(data)
	if err != nil {
		return "", err
	}
	frameRef := near.FrameRef{}
	err = frameRef.UnmarshalBinary(frameRefBytes)
	if err != nil {
		return nil, err
	}

	return hex.EncodeToString(frameRefBytes), nil
}

// GetBlock retrieves a block from Avail chain by block hash
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

func (c Client) keysetHash() ([32]byte, []byte, error) {
	svc := ServiceDetails{
		service:     (DataAvailabilityServiceWriter)(s),
		pubKey:      s.pub,
		signersMask: 1,
		metricName:  "near",
	}

	services := make([]ServiceDetails, 1)
	services[0] = svc
	return KeysetHashFromServices(services, 1)
}

func (s *NearService) GetByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	log.Info("Getting message", "hash", hash)
	// Hack to bypass commitment
	bytesPadded := make([]byte, 64)
	copy(bytesPadded[0:32], hash.Bytes())
	bytes, err := s.near.Get(bytesPadded, 0)
	if err != nil {
		return nil, err
	}
	return bytes, nil

}
