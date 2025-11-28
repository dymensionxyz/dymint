package bnb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/ethutils"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/dymensionxyz/go-ethereum"

	"github.com/dymensionxyz/go-ethereum/consensus/misc/eip4844"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"

	"github.com/dymensionxyz/go-ethereum/ethclient"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

type BNBClient interface {
	SubmitBlob(blob []byte) (common.Hash, []byte, []byte, error)
	GetBlob(txHash string) ([]byte, error)
	GetAccountAddress() string
	GetSignerBalance() (*big.Int, error)
	ValidateInclusion(txHash string, commitment []byte, proof []byte) error
}

type Client struct {
	ethclient *ethclient.Client
	rpcClient *rpc.Client
	ctx       context.Context
	cfg       *BNBConfig
	account   *Account
}

// ValidateInclusion validates that there is a blob included in the tx and corresponds to the commitment and proof included in da path.
func (c Client) ValidateInclusion(txHash string, txCommitment []byte, txProof []byte) error {
	// if blobsidecar not retrieved it may mean rpc error, therefore not considered da.ErrBlobNotFound
	blobSidecar, err := c.blobSidecarByTxHash(c.ctx, txHash)
	if err != nil {
		return err
	}

	// if there is no sidecar attached to tx it means the blob is not posted
	if blobSidecar == nil {
		return da.ErrBlobNotFound
	}

	for i, sidecarCommitment := range blobSidecar.Sidecar.Commitments {

		// if unable to unmarshall received commitment continue to next
		comm, err := sidecarCommitment.MarshalText()
		if err != nil {
			continue
		}

		// we pick the blob that matches with commitment in da path
		if bytes.Equal(comm, txCommitment) {

			// convert received 0x format proof to kzg4844.Proof
			var proof kzg4844.Proof
			err := proof.UnmarshalJSON(txProof)
			if err != nil {
				return errors.Join(da.ErrBlobNotFound, err)
			}

			// convert received 0x format commitment to  kzg4844.Commitment
			var commitment kzg4844.Commitment
			err = commitment.UnmarshalJSON(txCommitment)
			if err != nil {
				return errors.Join(da.ErrBlobNotFound, err)
			}

			// validate blob with da path commitment and proof
			err = ethutils.VerifyBlobProof((*ethutils.Blob)(blobSidecar.Sidecar.Blobs[i]), commitment, proof)
			if err != nil {
				return errors.Join(da.ErrBlobNotFound, err)
			}
			return nil
		}
	}

	return da.ErrBlobNotFound
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

// SubmitBlob sends blob data to BNB chain
func (c Client) SubmitBlob(blob []byte) (common.Hash, []byte, []byte, error) {
	// get nonce for the submitter account
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	nonce, err := c.ethclient.PendingNonceAt(cCtx, c.account.addr)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}

	// get base and tip fee from chain
	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(c.ctx)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}

	// computes the recommended gas fee with base and tip obtained
	gasFeeCap := ethutils.CalcGasFeeCap(baseFee, gasTipCap)

	// estimate gas using rpc
	cCtx, cancel = context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	to := common.HexToAddress(ArchivePoolAddress)
	msg := ethereum.CallMsg{
		From:      c.account.addr,
		To:        &to,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Data:      nil,
		Value:     nil,
	}
	gas, err := c.ethclient.EstimateGas(cCtx, msg)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}

	// create blob tx with blob and fee params previously obtained
	blobTx, err := createBlobTx(c.account.Key, c.cfg.ChainId, gas, gasTipCap, gasFeeCap, blobBaseFee, blob, common.HexToAddress(ArchivePoolAddress), nonce)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}

	// send blob tx to BNB chain
	cCtx, cancel = context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	err = c.ethclient.SendTransaction(cCtx, blobTx)
	if err != nil {
		return common.Hash{}, nil, nil, err
	}
	txhash := blobTx.Hash()

	// parse blob commitment generated to 0x format
	commitment, err := blobTx.BlobTxSidecar().Commitments[0].MarshalText()
	if err != nil {
		return common.Hash{}, nil, nil, err
	}

	// parse blob proof generated to 0x format
	proof, err := blobTx.BlobTxSidecar().Proofs[0].MarshalText()
	if err != nil {
		return common.Hash{}, nil, nil, err
	}
	return txhash, commitment, proof, nil
}

// GetBlob retrieves a blob from BNB chain by tx hash
func (c Client) GetBlob(txhash string) ([]byte, error) {
	blobSidecar, err := c.blobSidecarByTxHash(c.ctx, txhash)
	if err != nil {
		return nil, err
	}

	if blobSidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	var data []byte
	for _, blob := range blobSidecar.Sidecar.Blobs {
		b := (ethutils.Blob)(*blob)
		data, err = b.ToData()
		if err == nil {
			break
		}
	}
	if data == nil {
		return nil, fmt.Errorf("error recovering data from blob")
	}
	return data, err
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return c.account.addr.String()
}

// blobSidecarByTxHash return a sidecar of a given blob transaction
func (c Client) blobSidecarByTxHash(ctx context.Context, hash string) (*BlobSidecarTx, error) {
	var sidecar *BlobSidecarTx
	cCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	err := c.rpcClient.CallContext(cCtx, &sidecar, "eth_getBlobSidecarByTxHash", hash)
	if err != nil {
		return nil, err
	}
	if sidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	return sidecar, nil
}

// GetSignerBalance returns account balance
func (c Client) GetSignerBalance() (*big.Int, error) {
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	return c.ethclient.PendingBalanceAt(cCtx, c.account.addr)
}

// suggestGasPriceCaps suggests what the new tip, base fee, and blob base fee should be based on
// the current BNB chain conditions. blobfee will be nil if 4844 is not yet active.
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
