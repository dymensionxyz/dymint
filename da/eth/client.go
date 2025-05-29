package eth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/ethutils"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/go-ethereum/consensus/misc/eip4844"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"

	"github.com/dymensionxyz/go-ethereum/ethclient"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

const (
	beaconBlockUrl = "/eth/v2/beacon/blocks/"
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

// ValidateInclusion validates that there is a blob included in the beacon chain block and corresponds to the commitment and proof included in da path.
func (c Client) ValidateInclusion(slot string, txCommitment []byte, txProof []byte) error {
	blob, err := c.getKzg4844Blob(slot, txCommitment)
	if err != nil {
		return err
	}

	// convert received 0x format commitment to  kzg4844.Commitment
	var commitment kzg4844.Commitment
	err = commitment.UnmarshalJSON([]byte(fmt.Sprintf("%q", txCommitment)))
	if err != nil {
		return errors.Join(da.ErrBlobNotFound, err)
	}

	// convert received 0x format proof to kzg4844.Proof
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

	// get base and tip fee from chain
	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(c.ctx)
	if err != nil {
		return nil, nil, "", err
	}

	// computes the recommended gas fee with base and tip obtained
	gasFeeCap := calcGasFeeCap(baseFee, gasTipCap)

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

	receipt, err := c.waitForTxReceipt(txhash)
	if err != nil {
		return nil, nil, "", err
	}

	// parse blob commitment generated to 0x format
	commitment, err := blobTx.BlobTxSidecar().Commitments[0].MarshalText()
	if err != nil {
		return nil, nil, "", err
	}

	// parse blob proof generated to 0x format
	proof, err := blobTx.BlobTxSidecar().Proofs[0].MarshalText()
	if err != nil {
		return nil, nil, "", err
	}

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

func (c Client) getKzg4844Blob(slot string, txCommitment []byte) (*kzg4844.Blob, error) {
	blobSidecars, err := c.retrieveBlobSidecar(slot)
	if err != nil {
		return nil, err
	}

	if blobSidecars == nil {
		return nil, gerrc.ErrNotFound
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

// GetAccountAddress returns configured account address
func (c Client) GetAccountAddress() string {
	return c.account.addr.String()
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

// getSlot returns the beacon chain slot id that corresponds to the execution layer block id
func (c *Client) getSlot(blockId uint64) (string, error) {
	blockIdHead, slotHead, err := c.obtainSlotInfo("head")
	if err != nil {
		return "", err
	}

	blockDifference := blockIdHead - blockId
	slotTarget := slotHead - blockDifference

	for {
		block, slot, err := c.obtainSlotInfo(strconv.FormatUint(slotTarget, 10))
		if err != nil {
			return "", err
		}
		if block < blockId {
			return "", fmt.Errorf("this should not happen")
		}
		if block == blockId {
			return strconv.FormatUint(slot, 10), nil
		}
		blockDifference := block - blockId
		slotTarget = slotTarget - blockDifference
	}
}

// obtainSlotInfo returns block and slot id for certain slot (slot accepts either slot id or head string)
func (c *Client) obtainSlotInfo(slot string) (uint64, uint64, error) {
	url := c.apiURL + beaconBlockUrl + slot

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return uint64(0), uint64(0), fmt.Errorf("post failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return uint64(0), uint64(0), fmt.Errorf("unexpected status: %s, body: %s", resp.Status, body)
	}

	var bnResp BeaconChainResponse
	if err := json.NewDecoder(resp.Body).Decode(&bnResp); err != nil {
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
	url := fmt.Sprintf(c.apiURL+"/eth/v1/beacon/blob_sidecars/%s", slot)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("beacon chain http get failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected http status: %s, body: %s", resp.Status, body)
	}

	var blobResp BlobSidecarResponse
	if err := json.NewDecoder(resp.Body).Decode(&blobResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return blobResp.Data, nil
}

// GetSignerBalance returns account balance
func (c Client) GetSignerBalance() (*big.Int, error) {
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	return c.ethclient.PendingBalanceAt(cCtx, c.account.addr)
}

// suggestGasPriceCaps suggests what the new tip, base fee, and blob base fee should be based on
// the current  chain conditions.
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

	nextBaseFee := calculateNextBaseFee(head)

	var blobFee *big.Int
	if head.ExcessBlobGas != nil {
		blobFee = eip4844.CalcBlobFee(*head.ExcessBlobGas)
	}
	return tip, nextBaseFee, blobFee, nil
}

func calculateNextBaseFee(parent *types.Header) *big.Int {
	elasticityMultiplier := uint64(2)
	baseFeeMaxChangeDenominator := uint64(8)

	if parent.BaseFee == nil {
		return nil // Pre-EIP-1559 block
	}

	gasUsed := parent.GasUsed
	gasLimit := parent.GasLimit
	targetGas := gasLimit / elasticityMultiplier
	baseFee := new(big.Int).Set(parent.BaseFee)

	if gasUsed == targetGas {
		return baseFee
	}

	delta := int64(gasUsed - targetGas) //nolint:gosec // disable G115
	change := new(big.Int).Mul(baseFee, big.NewInt(delta))
	change.Div(change, big.NewInt(int64(targetGas)))                   //nolint:gosec // disable G115
	change.Div(change, big.NewInt(int64(baseFeeMaxChangeDenominator))) //nolint:gosec // disable G115

	nextBaseFee := new(big.Int)
	if gasUsed > targetGas {
		nextBaseFee.Add(baseFee, change)
	} else {
		nextBaseFee.Sub(baseFee, change)
	}

	if nextBaseFee.Cmp(big.NewInt(0)) < 0 {
		nextBaseFee = big.NewInt(0)
	}

	return nextBaseFee
}

/*func (c Client) estimateGasSimple(
	ctx context.Context,
	realData []byte,
	realBlobs []kzg4844.Blob,
	realAccessList types.AccessList,
) (uint64, error) {

	latestHeader, err := c.ethclient.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	maxFeePerGas := arbmath.BigMulByUBips(latestHeader.BaseFee, config.GasEstimateBaseFeeMultipleBips)
	_, realBlobHashes, err := blobs.ComputeCommitmentsAndHashes(realBlobs)
	if err != nil {
		return 0, fmt.Errorf("failed to compute real blob commitments: %w", err)
	}
	// If we're at the latest nonce, we can skip the special future tx estimate stuff
	gas, err := c.estimateGas(ctx, estimateGasParams{
		From:         c.account.addr,
		To:           c.dest,
		Data:         nil,
		MaxFeePerGas: (*hexutil.Big)(maxFeePerGas),
		BlobHashes:   realBlobHashes,
	}, "latest")
	if err != nil {
		return 0, fmt.Errorf("Error gas estimation: %w", err)
	}
	return gas + 50_000, nil
}

func (c Client) estimateGas(ctx context.Context, params estimateGasParams, blockHex string) (uint64, error) {
	var gas hexutil.Uint64
	err := c.rpcClient.CallContext(ctx, &gas, "eth_estimateGas", params, blockHex)
	return uint64(gas), err
}*/
