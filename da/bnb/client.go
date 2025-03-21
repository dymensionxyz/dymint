package bnb

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	rpcClient *rpc.Client
	ctx       context.Context
}

type Config struct {
	// The multiplier applied to fee suggestions to put a hard limit on fee increases.
	FeeLimitMultiplier uint64
	// Minimum threshold (in Wei) at which the FeeLimitMultiplier takes effect.
	// On low-fee networks, like test networks, this allows for arbitrary fee bumps
	// below this threshold.
	FeeLimitThreshold *big.Int
	// The maximum limit (in GWei) of blob gas price,
	// Above which will stop submit and wait for the price go down
	BlobGasPriceLimit *big.Int
	// Minimum base fee (in Wei) to assume when determining tx fees.
	MinBaseFee *big.Int
	// Minimum tip cap (in Wei) to enforce when determining tx fees.
	MinTipCap *big.Int
	// Signer is used to sign transactions when the gas price is increased.
	//Signer  opcrypto.SignerFn
	From    common.Address
	chainID *big.Int
}

var _ BNBClient = &Client{}

func NewClient(ctx context.Context, config BNBConfig) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	client := Client{
		rpcClient: rpcClient,
		ctx:       ctx,
	}

	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) ([]byte, error) {

	/*var b eth.Blob
	err := b.FromData(blob)
	if err != nil {
		return nil, err
	}
	sidecar, blobHashes, err := txmgr.MakeSidecar([]*eth.Blob{&b})
	require.NoError(t, err)
	require.NotNil(t, pendingHeader.ExcessBlobGas, "need L1 header with 4844 properties")
	blobBaseFee := eip4844.CalcBlobFee(*pendingHeader.ExcessBlobGas)
	blobFeeCap := new(uint256.Int).Mul(uint256.NewInt(2), uint256.MustFromBig(blobBaseFee))
	if blobFeeCap.Lt(uint256.NewInt(params.GWei)) { // ensure we meet 1 gwei geth tx-pool minimum
		blobFeeCap = uint256.NewInt(params.GWei)
	}
	txData = &types.BlobTx{
		To:         s.rollupCfg.BatchInboxAddress,
		Data:       nil,
		Gas:        params.TxGas, // intrinsic gas only
		BlobHashes: blobHashes,
		Sidecar:    sidecar,
		ChainID:    uint256.MustFromBig(s.rollupCfg.L1ChainID),
		GasTipCap:  uint256.MustFromBig(gasTipCap),
		GasFeeCap:  uint256.MustFromBig(gasFeeCap),
		BlobFeeCap: blobFeeCap,
		Value:      uint256.NewInt(0),
		Nonce:      nonce,
	}*/

	return nil, nil
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(frameRef []byte) ([]byte, error) {

	return nil, nil
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}

/*func (c *Client) sendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return c.rpcClient.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}*/
