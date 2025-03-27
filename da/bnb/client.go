package bnb

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/dymensionxyz/go-ethereum"

	"github.com/dymensionxyz/go-ethereum/consensus/misc/eip4844"

	"github.com/dymensionxyz/go-ethereum/ethclient"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) (common.Hash, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	ethclient *ethclient.Client
	rpcClient *rpc.Client
	ctx       context.Context
	cfg       *BNBConfig
	account   *Account
}

var _ BNBClient = &Client{}

func NewClient(ctx context.Context, config *BNBConfig) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	account, err := fromHexKey(config.PrivateKey)
	if err != nil {
		return nil, err
	}

	client := Client{
		ethclient: ethclient.NewClient(rpcClient),
		rpcClient: rpcClient,
		ctx:       ctx,
		cfg:       config,
		account:   account,
	}

	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) (common.Hash, error) {

	nonce, err := c.ethclient.PendingNonceAt(context.Background(), c.account.addr)
	if err != nil {
		return common.Hash{}, err
	}

	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(c.ctx)

	gasFeeCap := calcGasFeeCap(baseFee, gasTipCap)

	to := common.HexToAddress(ArchivePoolAddress)
	gas, err := c.ethclient.EstimateGas(c.ctx, ethereum.CallMsg{
		From:      c.account.addr,
		To:        &to,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      nil,
		Value:     nil,
	})

	blobTx, err := createBlobTx(c.account.Key, c.cfg.ChainId, gas, gasTipCap, gasFeeCap, blobBaseFee, blob, common.HexToAddress(ArchivePoolAddress), nonce)

	if err != nil {
		return common.Hash{}, err
	}

	err = c.ethclient.SendTransaction(context.Background(), blobTx)
	if err != nil {
		return common.Hash{}, err
	}
	txhash := blobTx.Hash()
	return txhash, nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(txhash string) ([]byte, error) {

	blobSidecar, err := c.BlobSidecarByTxHash(c.ctx, txhash)
	if err != nil {
		return nil, err
	}

	if blobSidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	var data []byte
	for _, blob := range blobSidecar.Sidecar.Blobs {
		b := (Blob)(*blob)
		data, err = b.ToData()
		if err == nil {
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("Error recovering data from blob")
	}
	return data, err
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return c.account.addr.String()
}

// BlobSidecarByTxHash return a sidecar of a given blob transaction
func (c Client) BlobSidecarByTxHash(ctx context.Context, hash string) (*BlobSidecarTx, error) {
	var sidecar *BlobSidecarTx
	err := c.rpcClient.CallContext(ctx, &sidecar, "eth_getBlobSidecarByTxHash", hash)
	if err != nil {
		return nil, err
	}
	if sidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	return sidecar, nil

}

// suggestGasPriceCaps suggests what the new tip, base fee, and blob base fee should be based on
// the current L1 conditions. blobfee will be nil if 4844 is not yet active.
func (c Client) suggestGasPriceCaps(ctx context.Context) (*big.Int, *big.Int, *big.Int, error) {
	cCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	tip, err := c.ethclient.SuggestGasTipCap(cCtx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested gas tip cap: %w", err)
	} else if tip == nil {
		return nil, nil, nil, errors.New("the suggested tip was nil")
	}
	cCtx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	head, err := c.ethclient.HeaderByNumber(cCtx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested base fee: %w", err)
	}

	// basefee of BSC block is 0
	baseFee := big.NewInt(0)

	var blobFee *big.Int
	if head.ExcessBlobGas != nil {
		blobFee = eip4844.CalcBlobFee(*head.ExcessBlobGas)
	}
	return tip, baseFee, blobFee, nil
}
