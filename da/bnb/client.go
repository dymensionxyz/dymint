package bnb

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	dmtypes "github.com/dymensionxyz/dymint/types"
	opcrypto "github.com/ethereum-optimism/optimism/op-service/crypto"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum/go-ethereum"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) ([]byte, error)
	GetBlob(frameRef []byte) ([]byte, error)
	GetAccountAddress() string
}

type Client struct {
	rpcClient *rpc.Client
	cfg       *Config
	ctx       context.Context
	nonce     *uint64
}

var (
	_            BNBClient = &Client{}
	two                    = big.NewInt(2)
	minBlobTxFee           = big.NewInt(params.GWei)
)

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
	Signer  opcrypto.SignerFn
	From    common.Address
	chainID *big.Int
}

// TxCandidate is a transaction candidate that can be submitted to ask the
// [TxManager] to construct a transaction with gas price bounds.
type TxCandidate struct {
	// TxData is the transaction calldata to be used in the constructed tx.
	TxData []byte
	// Blobs to send along in the tx (optional). If len(Blobs) > 0 then a blob tx
	// will be sent instead of a DynamicFeeTx.
	Blobs []*eth.Blob
	// To is the recipient of the constructed tx. Nil means contract creation.
	To *common.Address
	// GasLimit is the gas limit to be used in the constructed tx.
	GasLimit uint64
	// Value is the value to be used in the constructed tx.
	Value *big.Int
}

// NewClient returns a DA avail client
func NewClient(ctx context.Context, config BNBConfig, logger dmtypes.Logger) (BNBClient, error) {

	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	signerFactory, from, err := opcrypto.SignerFactoryFromConfig(nil, config.PrivateKey, "", "", opsigner.NewCLIConfig())
	if err != nil {
		return nil, fmt.Errorf("could not init signer: %w", err)
	}

	feeLimitThreshold, err := eth.GweiToWei(config.FeeLimitThresholdGwei)
	if err != nil {
		return nil, fmt.Errorf("invalid fee limit threshold: %w", err)
	}

	blobGasPriceLimit, err := eth.GweiToWei(config.BlobGasPriceLimitGwei)
	if err != nil {
		return nil, fmt.Errorf("invalid blob gas price limit: %w", err)
	}

	minBaseFee, err := eth.GweiToWei(config.MinBaseFeeGwei)
	if err != nil {
		return nil, fmt.Errorf("invalid min base fee: %w", err)
	}

	minTipCap, err := eth.GweiToWei(config.MinTipCapGwei)
	if err != nil {
		return nil, fmt.Errorf("invalid min tip cap: %w", err)
	}
	cfg := &Config{
		FeeLimitThreshold: feeLimitThreshold,
		BlobGasPriceLimit: blobGasPriceLimit,
		MinBaseFee:        minBaseFee,
		MinTipCap:         minTipCap,
		From:              from,
	}

	client := Client{
		rpcClient: rpcClient,
		cfg:       cfg,
		ctx:       ctx,
	}

	chainID, err := client.getChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not dial fetch L1 chain ID: %w", err)
	}
	cfg.chainID = chainID
	cfg.Signer = signerFactory(chainID)
	return client, nil
}

// SubmitData sends blob data to Avail DA
func (c Client) SubmitBlob(blob []byte) ([]byte, error) {

	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	err = c.rpcClient.CallContext(c.ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
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

// craftTx creates the signed transaction
// It queries L1 for the current fee market conditions as well as for the nonce.
// NOTE: This method SHOULD NOT publish the resulting transaction.
// NOTE: If the [TxCandidate.GasLimit] is non-zero, it will be used as the transaction's gas.
// NOTE: Otherwise, the [SimpleTxManager] will query the specified backend for an estimate.
func (c *Client) craftTx(ctx context.Context, candidate TxCandidate) (*types.Transaction, error) {
	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price info: %w", err)
	}
	gasFeeCap := calcGasFeeCap(baseFee, gasTipCap)

	gasLimit := candidate.GasLimit

	// If the gas limit is set, we can use that as the gas
	if gasLimit == 0 {
		// Calculate the intrinsic gas for the transaction
		gas, err := c.estimateGas(ctx, ethereum.CallMsg{
			From:      c.cfg.From,
			To:        candidate.To,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Data:      candidate.TxData,
			Value:     candidate.Value,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to estimate gas: %w", err)
		}
		gasLimit = gas
	}

	var sidecar *types.BlobTxSidecar
	var blobHashes []common.Hash
	if len(candidate.Blobs) > 0 {
		if candidate.To == nil {
			return nil, errors.New("blob txs cannot deploy contracts")
		}
		if sidecar, blobHashes, err = MakeSidecar(candidate.Blobs); err != nil {
			return nil, fmt.Errorf("failed to make sidecar: %w", err)
		}
	}

	var txMessage types.TxData
	if sidecar != nil {
		if blobBaseFee == nil {
			return nil, fmt.Errorf("expected non-nil blobBaseFee")
		}
		// no need to calcBlobFeeCap, prefer raw blobBaseFee
		// blobFeeCap := calcBlobFeeCap(blobBaseFee)
		blobFeeCap := blobBaseFee
		message := &types.BlobTx{
			To:         *candidate.To,
			Data:       candidate.TxData,
			Gas:        gasLimit,
			BlobHashes: blobHashes,
			Sidecar:    sidecar,
		}
		if err := finishBlobTx(message, c.cfg.chainID, gasTipCap, gasFeeCap, blobFeeCap, candidate.Value); err != nil {
			return nil, fmt.Errorf("failed to create blob transaction: %w", err)
		}
		txMessage = message
	} else {
		txMessage = &types.DynamicFeeTx{
			ChainID:   c.cfg.chainID,
			To:        candidate.To,
			GasTipCap: gasTipCap,
			GasFeeCap: gasFeeCap,
			Value:     candidate.Value,
			Data:      candidate.TxData,
			Gas:       gasLimit,
		}
	}
	return m.signWithNextNonce(ctx, txMessage) // signer sets the nonce field of the tx
}

// calcGasFeeCap deterministically computes the recommended gas fee cap given
// the base fee and gasTipCap. The resulting gasFeeCap is equal to:
//
//	gasTipCap + 2*baseFee.
func calcGasFeeCap(baseFee, gasTipCap *big.Int) *big.Int {
	return new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(baseFee, two),
	)
}

// calcBlobFeeCap computes a suggested blob fee cap that is twice the current header's blob base fee
// value, with a minimum value of minBlobTxFee.
func calcBlobFeeCap(blobBaseFee *big.Int) *big.Int {
	cap := new(big.Int).Mul(blobBaseFee, two)
	if cap.Cmp(minBlobTxFee) < 0 {
		cap.Set(minBlobTxFee)
	}
	return cap
}

// suggestGasPriceCaps suggests what the new tip, base fee, and blob base fee should be based on
// the current L1 conditions. blobfee will be nil if 4844 is not yet active.
func (c *Client) suggestGasPriceCaps(ctx context.Context) (*big.Int, *big.Int, *big.Int, error) {
	tip, err := c.suggestGasPrice(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested gas tip cap: %w", err)
	} else if tip == nil {
		return nil, nil, nil, errors.New("the suggested tip was nil")
	}

	head, err := c.headerByNumber(ctx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested base fee: %w", err)
	}

	// basefee of BSC block is 0
	baseFee := big.NewInt(0)

	// Enforce minimum base fee and tip cap
	if minTipCap := c.cfg.MinTipCap; minTipCap != nil && tip.Cmp(minTipCap) == -1 {
		tip = new(big.Int).Set(c.cfg.MinTipCap)
	}
	if minBaseFee := c.cfg.MinBaseFee; minBaseFee != nil && baseFee.Cmp(minBaseFee) == -1 {
		baseFee = new(big.Int).Set(c.cfg.MinBaseFee)
	}

	var blobFee *big.Int
	if head.ExcessBlobGas != nil {
		blobFee = eip4844.CalcBlobFee(params.TestChainConfig, head)
	}
	return tip, baseFee, blobFee, nil
}

// SuggestGasPrice retrieves the currently suggested gas price to allow a timely
// execution of a transaction.
func (c *Client) suggestGasPrice(ctx context.Context) (*big.Int, error) {
	var hex hexutil.Big
	if err := c.rpcClient.CallContext(ctx, &hex, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return (*big.Int)(&hex), nil
}

// HeaderByNumber returns a block header from the current canonical chain. If number is
// nil, the latest known header is returned.
func (c *Client) headerByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := c.rpcClient.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return head, err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	// It's negative.
	if number.IsInt64() {
		return rpc.BlockNumber(number.Int64()).String()
	}
	// It's negative and large, which is invalid.
	return fmt.Sprintf("<invalid %d>", number)
}

// EstimateGas tries to estimate the gas needed to execute a specific transaction based on
// the current pending state of the backend blockchain. There is no guarantee that this is
// the true gas limit requirement as other transactions may be added or removed by miners,
// but it should provide a basis for setting a reasonable default.
func (c *Client) estimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	var hex hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &hex, "eth_estimateGas", toCallArg(msg))
	if err != nil {
		return 0, err
	}
	return uint64(hex), nil
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["input"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.GasFeeCap != nil {
		arg["maxFeePerGas"] = (*hexutil.Big)(msg.GasFeeCap)
	}
	if msg.GasTipCap != nil {
		arg["maxPriorityFeePerGas"] = (*hexutil.Big)(msg.GasTipCap)
	}
	if msg.AccessList != nil {
		arg["accessList"] = msg.AccessList
	}
	if msg.BlobGasFeeCap != nil {
		arg["maxFeePerBlobGas"] = (*hexutil.Big)(msg.BlobGasFeeCap)
	}
	if msg.BlobHashes != nil {
		arg["blobVersionedHashes"] = msg.BlobHashes
	}
	return arg
}

// signWithNextNonce returns a signed transaction with the next available nonce.
// The nonce is fetched once using eth_getTransactionCount with "latest", and
// then subsequent calls simply increment this number. If the transaction manager
// is reset, it will query the eth_getTransactionCount nonce again. If signing
// fails, the nonce is not incremented.
func (c *Client) signWithNextNonce(ctx context.Context, txMessage types.TxData) (*types.Transaction, error) {

	if c.nonce == nil {

		nonce, err := c.nonceAt(ctx, c.cfg.From, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get nonce: %w", err)
		}
		c.nonce = &nonce
	} else {
		*c.nonce++
	}

	switch x := txMessage.(type) {
	case *types.DynamicFeeTx:
		x.Nonce = *c.nonce
	case *types.BlobTx:
		x.Nonce = *c.nonce
	default:
		return nil, fmt.Errorf("unrecognized tx type: %T", x)
	}

	tx, err := c.cfg.Signer(ctx, c.cfg.From, types.NewTx(txMessage))
	if err != nil {
		// decrement the nonce, so we can retry signing with the same nonce next time
		// signWithNextNonce is called
		*c.nonce--
	}
	return tx, err
}

// NonceAt returns the account nonce of the given account.
// The block number can be nil, in which case the nonce is taken from the latest known block.
func (c *Client) nonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &result, "eth_getTransactionCount", account, toBlockNumArg(blockNumber))
	return uint64(result), err
}

// ChainID retrieves the current chain ID for transaction replay protection.
func (c *Client) getChainID(ctx context.Context) (*big.Int, error) {
	var result hexutil.Big
	err := c.rpcClient.CallContext(ctx, &result, "eth_chainId")
	if err != nil {
		return nil, err
	}
	return (*big.Int)(&result), err
}

// MakeSidecar builds & returns the BlobTxSidecar and corresponding blob hashes from the raw blob
// data.
func MakeSidecar(blobs []*eth.Blob) (*types.BlobTxSidecar, []common.Hash, error) {
	sidecar := &types.BlobTxSidecar{}
	blobHashes := make([]common.Hash, 0, len(blobs))
	for i, blob := range blobs {
		rawBlob := *blob.KZGBlob()
		sidecar.Blobs = append(sidecar.Blobs, rawBlob)
		commitment, err := kzg4844.BlobToCommitment(rawBlob)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot compute KZG commitment of blob %d in tx candidate: %w", i, err)
		}
		sidecar.Commitments = append(sidecar.Commitments, commitment)
		proof, err := kzg4844.ComputeBlobProof(rawBlob, commitment)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot compute KZG proof for fast commitment verification of blob %d in tx candidate: %w", i, err)
		}
		sidecar.Proofs = append(sidecar.Proofs, proof)
		blobHashes = append(blobHashes, eth.KZGToVersionedHash(commitment))
	}
	return sidecar, blobHashes, nil
}

// finishBlobTx finishes creating a blob tx message by safely converting bigints to uint256
func finishBlobTx(message *types.BlobTx, chainID, tip, fee, blobFee, value *big.Int) error {
	var o bool
	if message.ChainID, o = uint256.FromBig(chainID); o {
		return fmt.Errorf("ChainID overflow")
	}
	if message.GasTipCap, o = uint256.FromBig(tip); o {
		return fmt.Errorf("GasTipCap overflow")
	}
	if message.GasFeeCap, o = uint256.FromBig(fee); o {
		return fmt.Errorf("GasFeeCap overflow")
	}
	if message.BlobFeeCap, o = uint256.FromBig(blobFee); o {
		return fmt.Errorf("BlobFeeCap overflow")
	}
	if message.Value, o = uint256.FromBig(value); o {
		return fmt.Errorf("Value overflow")
	}
	return nil
}
