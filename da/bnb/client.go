package bnb

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
)

type BNBClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	rpcClient *rpc.Client
	ctx       context.Context
	cfg       *Config
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
	From       common.Address
	To         common.Address
	ChainId    *big.Int
	PrivateKey string
}

var _ BNBClient = &Client{}

func NewClient(ctx context.Context, config BNBConfig) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}
	cfg := &Config{
		From:       config.From,
		To:         config.To,
		ChainId:    new(big.Int).SetUint64(config.ChainId),
		PrivateKey: config.PrivateKey,
	}
	client := Client{
		rpcClient: rpcClient,
		ctx:       ctx,
		cfg:       cfg,
	}

	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) ([]byte, error) {

	var b eth.Blob
	err := b.FromData(blob)
	if err != nil {
		return nil, err
	}
	sidecar, blobHashes, err := txmgr.MakeSidecar([]*eth.Blob{&b})
	if err != nil {
		return nil, err
	}
	blobFeeCap := uint256.NewInt(params.GWei)
	gasTipCap := uint256.NewInt(params.GWei)
	gasFeeCap := uint256.NewInt(params.GWei)
	nonce, err := c.PendingNonceAt(c.ctx, c.cfg.From)
	if err != nil {
		return nil, err
	}
	txData := &types.BlobTx{
		To:         c.cfg.To,
		Data:       nil,
		Gas:        params.TxGas, // intrinsic gas only
		BlobHashes: blobHashes,
		Sidecar:    sidecar,
		ChainID:    uint256.MustFromBig(c.cfg.ChainId),
		GasTipCap:  gasTipCap,
		GasFeeCap:  gasFeeCap,
		BlobFeeCap: blobFeeCap,
		Value:      uint256.NewInt(0),
		Nonce:      nonce,
	}
	signer := types.NewCancunSigner(c.cfg.ChainId)

	pKeyBytes, err := hexutil.Decode("0x" + c.cfg.PrivateKey)
	if err != nil {
		return nil, err
	}
	// Convert the private key bytes to an ECDSA private key.
	ecdsaPrivateKey, err := crypto.ToECDSA(pKeyBytes)
	if err != nil {
		return nil, err
	}

	tx, err := types.SignNewTx(ecdsaPrivateKey, signer, txData)

	err = c.sendTransaction(c.ctx, tx)
	return nil, err
}

// GetBlock retrieves a block from Near chain by block hash
func (c Client) GetBlob(frameRef []byte) ([]byte, error) {
	/*var txSidecars []*BSCBlobTxSidecar
	number := rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockID))
	err := c.rpcClient.CallContext(ctx, &txSidecars, "eth_getBlobSidecars", number.String())
	if err != nil {
		return nil, err
	}
	if txSidecars == nil {
		return nil, ethereum.NotFound
	}
	idx := 0
	for _, txSidecar := range txSidecars {
		txIndex, err := util.HexToUint64(txSidecar.TxIndex)
		if err != nil {
			return nil, err
		}
		for j := range txSidecar.BlobSidecar.Blobs {
			sidecars = append(sidecars,
				&types2.GeneralSideCar{
					Sidecar: structs.Sidecar{
						Index:         strconv.Itoa(idx),
						Blob:          txSidecar.BlobSidecar.Blobs[j],
						KzgCommitment: txSidecar.BlobSidecar.Commitments[j],
						KzgProof:      txSidecar.BlobSidecar.Proofs[j],
					},
					TxIndex: int64(txIndex),
					TxHash:  txSidecar.TxHash,
				},
			)
			idx++
		}
	}
	return sidecars, err*/
	return nil, nil
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return ""
}

// PendingNonceAt returns the account nonce of the given account in the pending state.
// This is the nonce that should be used for the next transaction.
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionCount", account, "pending")
	return uint64(result), err
}

func (c *Client) sendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return c.rpcClient.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}
