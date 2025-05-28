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
	"strconv"

	"github.com/avast/retry-go/v4"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dymensionxyz/go-ethereum/consensus/misc/eip4844"
	"github.com/dymensionxyz/go-ethereum/crypto/kzg4844"

	"github.com/dymensionxyz/go-ethereum/ethclient"

	"github.com/dymensionxyz/go-ethereum/common"
	"github.com/dymensionxyz/go-ethereum/core/types"
	"github.com/dymensionxyz/go-ethereum/rpc"
)

type EthClient interface {
	SubmitBlob(blob []byte) (common.Hash, []byte, []byte, string, error)
	GetBlob(txHash string, slot string, txCommitment []byte) ([]byte, error)
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

// ValidateInclusion validates that there is a blob included in the tx and corresponds to the commitment and proof included in da path.
func (c Client) ValidateInclusion(txHash string, txCommitment []byte, txProof []byte) error {
	// if blobsidecar not retrieved it may mean rpc error, therefore not considered da.ErrBlobNotFound
	blobSidecar, err := c.blobSidecarByTxHash(c.ctx, txHash, 0)
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
			err = VerifyBlobProof((*Blob)(blobSidecar.Sidecar.Blobs[i]), commitment, proof)
			if err != nil {
				return errors.Join(da.ErrBlobNotFound, err)
			}
			return nil
		}
	}

	return da.ErrBlobNotFound
}

var _ EthClient = &Client{}

func NewClient(ctx context.Context, config *EthConfig) (EthClient, error) {
	rpcClient, err := rpc.DialContext(ctx, config.Endpoint)
	if err != nil {
		return nil, err
	}

	account, err := fromHexKey(config.PrivateKey)
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

// SubmitBlob sends blob data to BNB chain
func (c Client) SubmitBlob(blob []byte) (common.Hash, []byte, []byte, string, error) {
	// get nonce for the submitter account
	cCtx, cancel := context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	nonce, err := c.ethclient.PendingNonceAt(cCtx, c.account.addr)
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	// get base and tip fee from chain
	gasTipCap, baseFee, blobBaseFee, err := c.suggestGasPriceCaps(c.ctx)
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	// computes the recommended gas fee with base and tip obtained
	gasFeeCap := calcGasFeeCap(baseFee, gasTipCap)

	// estimate gas using rpc
	/*cCtx, cancel = context.WithTimeout(c.ctx, c.cfg.Timeout)
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
	}*/

	gasLimit := uint64(21000) // Adjust as needed

	// create blob tx with blob and fee params previously obtained
	//blobTx, err := createBlobTx(c.account.Key, c.cfg.ChainId, gas, gasTipCap, gasFeeCap, blobBaseFee, blob, common.HexToAddress(ArchivePoolAddress), nonce)
	blobTx, err := createBlobTx(c.account.Key, c.cfg.ChainId, gasLimit, gasTipCap, gasFeeCap, blobBaseFee, blob, common.HexToAddress(ArchivePoolAddress), nonce)
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	// send blob tx to BNB chain
	cCtx, cancel = context.WithTimeout(c.ctx, c.cfg.Timeout)
	defer cancel()
	err = c.ethclient.SendTransaction(cCtx, blobTx)
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}
	txhash := blobTx.Hash()

	receipt, err := c.waitForTxReceipt(txhash)
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	// parse blob commitment generated to 0x format
	commitment, err := blobTx.BlobTxSidecar().Commitments[0].MarshalText()
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	// parse blob proof generated to 0x format
	proof, err := blobTx.BlobTxSidecar().Proofs[0].MarshalText()
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	slot, err := c.getSlot(receipt.BlockNumber.Uint64())
	if err != nil {
		return common.Hash{}, nil, nil, "", err
	}

	return txhash, commitment, proof, slot, nil
}

// GetBlock retrieves a block BNB Near chain by tx hash
func (c Client) GetBlob(txhash string, slot string, txCommitment []byte) ([]byte, error) {
	/*blobSidecar, err := c.blobSidecarByTxHash(c.ctx, txhash, slot)
	if err != nil {
		return nil, err
	}*/

	blobSidecar, err := c.retrieveBlob(slot)
	if err != nil {
		return nil, err
	}

	if blobSidecar == nil {
		return nil, gerrc.ErrNotFound
	}

	var data []byte
	for _, blobsidecar := range blobSidecar {
		comm, err := blobsidecar.Commitment.MarshalText()
		if err != nil {
			continue
		}
		if !bytes.Equal(comm, txCommitment) {
			continue
		}
		b := (Blob)(*blobsidecar.Blob)
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
			fmt.Println(receipt.BlockHash)
			if receipt.BlockNumber == nil || receipt.BlockNumber.Cmp(big.NewInt(0)) == 0 {
				return fmt.Errorf("no block number in receipt")
			}
			return nil
		},
		retry.Context(ctx),
		retry.Attempts(5), //nolint:gosec // RetryAttempts should be always positive
		retry.Delay(0),
		retry.DelayType(retry.FixedDelay), // Force fixed delay between attempts
		retry.LastErrorOnly(true),         // Only log the last error
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	return receipt, nil
}

// blobSidecarByTxHash return a sidecar of a given blob transaction
func (c Client) blobSidecarByTxHash(ctx context.Context, hash string, slot int) (*BlobSidecarTx, error) {
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

// getSlot
func (c *Client) getSlot(blockId uint64) (string, error) {

	blockIdHead, slotHead, err := c.retrieveSlot("head")
	if err != nil {
		return "", err
	}

	blockDifference := blockIdHead - blockId
	slotTarget := slotHead - blockDifference

	for {
		block, slot, err := c.retrieveSlot(strconv.FormatUint(slotTarget, 10))
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

// retrieveSlot
func (c *Client) retrieveSlot(slot string) (uint64, uint64, error) {
	url := fmt.Sprintf(c.apiURL + "/eth/v2/beacon/blocks/" + slot)

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

// retrieveBlobTx gets Tx, that includes blob parts, using  Kaspa REST-API server (https://api.kaspa.org/docs)
func (c *Client) retrieveBlob(slot string) ([]BlobSidecarTest, error) {
	url := fmt.Sprintf(c.apiURL+"/eth/v1/beacon/blob_sidecars/%s", slot)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("post failed: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			return
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status: %s, body: %s", resp.Status, body)
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

	delta := gasUsed - targetGas
	change := new(big.Int).Mul(baseFee, big.NewInt(int64(delta)))
	change.Div(change, big.NewInt(int64(targetGas)))
	change.Div(change, big.NewInt(int64(baseFeeMaxChangeDenominator)))

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
