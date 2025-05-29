package eth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/ethutils"

	"github.com/dymensionxyz/go-ethereum/consensus/misc/eip4844"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"

	"github.com/dymensionxyz/go-ethereum/ethclient"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

const (
	beaconBlockUrl = "%s/eth/v2/beacon/blocks/%s"
	blobSidecarUrl = "%s/eth/v1/beacon/blob_sidecars/%s"
)

type EthClient interface {
	SubmitBlob(blob []byte) ([]byte, []byte, string, error)
	GetBlob(slot string, txCommitment []byte) ([]byte, error)
	GetAccountAddress() string
	GetSignerBalance() (*big.Int, error)
	ValidateInclusion(txHash string, commitment []byte, proof []byte) error
}

type Client struct {
	ethclient  *ethclient.Client
	rpcClient  *rpc.Client
	httpClient *http.Client
	ctx        context.Context
	cfg        *EthConfig
	account    *Account
	apiURL     string
}

var _ EthClient = &Client{}

func NewClient(ctx context.Context, config *EthConfig) (EthClient, error) {
	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	priKeyHex := os.Getenv(config.PrivateKeyEnv)
	if priKeyHex == "" {
		return nil, fmt.Errorf("private key environment %s is not set or empty", config.PrivateKeyEnv)
	}

	account, err := fromHexKey(priKeyHex)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	client := Client{
		ethclient:  ethclient.NewClient(rpcClient),
		rpcClient:  rpcClient,
		httpClient: httpClient,
		ctx:        ctx,
		cfg:        config,
		account:    account,
		apiURL:     config.ApiUrl,
	}

	return client, nil
}

// SubmitBlob sends blob data to Ethereum
func (c Client) SubmitBlob(blob []byte) ([]byte, []byte, string, error) {
	// get nonce for the submitter account
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	nonce, err := c.ethclient.PendingNonceAt(cCtx, c.account.addr)
	if err != nil {
		return nil, nil, "", err
	}

	// get base (regular and blob) and tip fee from chain
	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(c.ctx)
	if err != nil {
		return nil, nil, "", err
	}

	// computes the recommended gas fee with base and tip obtained
	gasFeeCap := ethutils.CalcGasFeeCap(baseFee, gasTipCap)

	// create blob tx with blob and fee params previously obtained
	blobTx, err := createBlobTx(c.account.Key, c.cfg.ChainId, *c.cfg.GasLimit, gasTipCap, gasFeeCap, blobBaseFee, blob, common.HexToAddress(ArchivePoolAddress), nonce)
	if err != nil {
		return nil, nil, "", err
	}

	// send blob tx to Ethereum
	cCtx, cancel = context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	err = c.ethclient.SendTransaction(cCtx, blobTx)
	if err != nil {
		return nil, nil, "", err
	}
	txhash := blobTx.Hash()

	// wait for inclusion receipt
	receipt, err := c.waitForTxReceipt(txhash)
	if err != nil {
		return nil, nil, "", err
	}

	// parse blob commitment generated to hex format
	commitment, err := blobTx.BlobTxSidecar().Commitments[0].MarshalText()
	if err != nil {
		return nil, nil, "", err
	}

	// parse blob proof generated to hex format
	proof, err := blobTx.BlobTxSidecar().Proofs[0].MarshalText()
	if err != nil {
		return nil, nil, "", err
	}

	// retrieve slot number of the beacon chain where the blob is included
	slot, err := c.getSlot(receipt.BlockNumber.Uint64())
	if err != nil {
		return nil, nil, "", err
	}

	return commitment, proof, slot, nil
}

// GetBlock retrieves a block Ethereum beacon chain by tx hash
func (c Client) GetBlob(slot string, txCommitment []byte) ([]byte, error) {
	blob, err := c.getKzg4844Blob(slot, txCommitment)
	if err != nil {
		return nil, err
	}
	b := (ethutils.Blob)(*blob)
	data, err := b.ToData()
	if err != nil {
		return nil, err
	}
	return data, nil
}

// ValidateInclusion validates that there is a blob included in the beacon chain block and corresponds to the commitment and proof included in da path.
func (c Client) ValidateInclusion(slot string, txCommitment []byte, txProof []byte) error {
	blob, err := c.getKzg4844Blob(slot, txCommitment)
	if err != nil {
		return err
	}

	// convert received hex format commitment to kzg4844.Commitment
	var commitment kzg4844.Commitment
	err = commitment.UnmarshalJSON([]byte(fmt.Sprintf("%q", txCommitment)))
	if err != nil {
		return errors.Join(da.ErrBlobNotFound, err)
	}

	// convert received hex format proof to kzg4844.Proof
	var proof kzg4844.Proof
	err = proof.UnmarshalJSON([]byte(fmt.Sprintf("%q", txProof)))
	if err != nil {
		return errors.Join(da.ErrBlobNotFound, err)
	}

	// validate blob with da path commitment and proof
	err = ethutils.VerifyBlobProof((*ethutils.Blob)(blob), commitment, proof)
	if err != nil {
		return errors.Join(da.ErrBlobNotFound, err)
	}
	return nil
}

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return c.account.addr.String()
}

// GetSignerBalance returns account balance
func (c Client) GetSignerBalance() (*big.Int, error) {
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	return c.ethclient.PendingBalanceAt(cCtx, c.account.addr)
}

func (c Client) waitForTxReceipt(txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	var receipt *types.Receipt
	err := retry.Do(
		func() error {
			var err error
			receipt, err = c.ethclient.TransactionReceipt(ctx, txHash)
			if err != nil {
				// Mark network/temporary errors as retryable
				return fmt.Errorf("get receipt failed: %w", err)
			}
			if receipt == nil {
				// Receipt not found yet - this is retryable
				return fmt.Errorf("receipt not found")
			}
			if receipt.BlockNumber == nil || receipt.BlockNumber.Cmp(big.NewInt(0)) == 0 {
				return fmt.Errorf("no block number in receipt")
			}
			return nil
		},
		retry.Context(ctx),
		retry.Attempts(10), //nolint:gosec // RetryAttempts should be always positive
		retry.Delay(1*time.Second),
		retry.DelayType(retry.FixedDelay), // Force fixed delay between attempts
		retry.LastErrorOnly(true),         // Only log the last error
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return receipt, nil
}

func (c Client) getKzg4844Blob(slot string, txCommitment []byte) (*kzg4844.Blob, error) {
	blobSidecars, err := c.retrieveBlobSidecar(slot)
	if err != nil {
		return nil, err
	}

	if blobSidecars == nil {
		return nil, errors.Join(da.ErrBlobNotFound, fmt.Errorf("Unable to get blob. No blobs included in slot id:%s", slot))
	}

	for _, blobSidecar := range blobSidecars {
		comm, err := blobSidecar.Commitment.MarshalText()
		if err != nil {
			continue
		}
		if bytes.Equal(comm, txCommitment) {
			return blobSidecar.Blob, nil
		}
	}
	return nil, errors.Join(da.ErrBlobNotFound, fmt.Errorf("Unable to get blob. No commitment match"))
}

// getSlot returns the beacon chain slot id that corresponds to the execution layer block id
func (c *Client) getSlot(blockId uint64) (string, error) {

	// to find the corresponding slot we start by getting the current head slot number
	blockIdHead, slotHead, err := c.obtainSlotInfo("head")
	if err != nil {
		return "", err
	}

	// then we calculate the blocks between head and block target
	blockDifference := blockIdHead - blockId
	// and we start searching with the slot with the same difference
	slotTarget := slotHead - blockDifference

	for {
		// retrieve block for the slot target
		block, _, err := c.obtainSlotInfo(strconv.FormatUint(slotTarget, 10))
		if err != nil {
			return "", err
		}
		// if block obtained is previous to target something wrong happened (there can be more slots that blocks, but not the opposite)
		if block < blockId {
			return "", fmt.Errorf("block retrieved should not be less than target block. block:%d target:%d slot:%d", block, blockId, slotTarget)
		}
		// return slot id when the block obtained is the target one
		if block == blockId {
			return strconv.FormatUint(slotTarget, 10), nil
		}

		//in case it does not match, we continue the search removing the block difference to slot target
		blockDifference := block - blockId
		slotTarget = slotTarget - blockDifference
	}
}

// obtainSlotInfo returns block and slot id for certain slot (slot accepts either slot id or head string)
func (c *Client) obtainSlotInfo(slot string) (uint64, uint64, error) {
	url := fmt.Sprintf(beaconBlockUrl, c.apiURL, slot)
	body, err := httpGet(url, c.httpClient)
	if err != nil {
		return uint64(0), uint64(0), err
	}

	var bnResp BeaconChainResponse
	if err := json.NewDecoder(body).Decode(&bnResp); err != nil {
		return uint64(0), uint64(0), fmt.Errorf("failed to decode response: %w", err)
	}
	blockId, err := strconv.ParseUint(bnResp.Data.Message.Body.ExecutionPayload.BlockNumber, 10, 64)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	currentSlot, err := strconv.ParseUint(bnResp.Data.Message.Slot, 10, 64)
	if err != nil {
		return uint64(0), uint64(0), err
	}
	return blockId, currentSlot, nil
}

// retrieveBlobSidecar returns blobsiders included in a Ethereum beacon chain block
func (c *Client) retrieveBlobSidecar(slot string) ([]BlobSidecar, error) {
	url := fmt.Sprintf(blobSidecarUrl, c.apiURL, slot)

	body, err := httpGet(url, c.httpClient)

	if err != nil {
		return nil, fmt.Errorf("failed to get response: %w", err)
	}
	var blobResp BlobSidecarResponse
	if err := json.NewDecoder(body).Decode(&blobResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return blobResp.Data, nil
}

// suggestGasPriceCaps suggests what the new tip, base fee, and blob base fee should be based on
// the current  chain conditions.
func (c Client) suggestGasPriceCaps(ctx context.Context) (*big.Int, *big.Int, *big.Int, error) {

	// suggested gas tip from chain
	cCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	tip, err := c.ethclient.SuggestGasTipCap(cCtx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested gas tip cap: %w", err)
	} else if tip == nil {
		return nil, nil, nil, errors.New("the suggested tip was nil")
	}

	// retrieve head header used for base fee calculation
	cCtx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()
	head, err := c.ethclient.HeaderByNumber(cCtx, nil)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch the suggested base fee: %w", err)
	}

	//eip-1559 base fee calculation
	nextBaseFee := calculateNextBaseFee(head)

	//eip-4844 blob base fee calculation
	var blobFee *big.Int
	if head.ExcessBlobGas != nil {
		blobFee = eip4844.CalcBlobFee(*head.ExcessBlobGas)
	}
	return tip, nextBaseFee, blobFee, nil
}
