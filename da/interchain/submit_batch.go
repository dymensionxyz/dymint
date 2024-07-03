package interchain

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	"github.com/avast/retry-go/v4"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

func (c *DALayerClient) SubmitBatch(*types.Batch) da.ResultSubmitBatch {
	panic("SubmitBatch method is not supported by the interchain DA clint")
}

func (c *DALayerClient) SubmitBatchV2(batch *types.Batch) da.ResultSubmitBatchV2 {
	commitment, err := c.submitBatch(batch)
	if err != nil {
		return da.ResultSubmitBatchV2{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("can't submit batch to the interchain DA layer: %s", err.Error()),
				Error:   err,
			},
			DAPath: da.Path{}, // empty in the error resp
		}
	}

	rawCommitment, err := cdctypes.NewAnyWithValue(commitment)
	if err != nil {
		return da.ResultSubmitBatchV2{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("can't submit batch to the interchain DA layer: %s", err.Error()),
				Error:   err,
			},
			DAPath: da.Path{}, // empty in the error resp
		}
	}

	// TODO: add MsgUpdateClint for DA<->Hub IBC client.

	return da.ResultSubmitBatchV2{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Submission successful",
		},
		DAPath: da.Path{
			DaType:     string(c.GetClientType()),
			Commitment: rawCommitment,
		},
	}
}

// submitBatch is used to process and transmit batches to the interchain DA.
func (c *DALayerClient) submitBatch(batch *types.Batch) (*interchainda.Commitment, error) {
	blob, err := EncodeBatch(batch)
	if err != nil {
		return nil, fmt.Errorf("can't encode batch to interchain DA format: %w", err)
	}

	// Verify the size of the blob is within the limit
	if len(blob) > int(c.daConfig.DAParams.MaxBlobSize) {
		return nil, fmt.Errorf("blob size %d exceeds the maximum allowed size %d", len(blob), c.daConfig.DAParams.MaxBlobSize)
	}

	// Calculate the fees of submitting this blob
	feesToPay := sdk.NewCoin(c.daConfig.DAParams.CostPerByte.Denom, c.daConfig.DAParams.CostPerByte.Amount.MulRaw(int64(len(blob))))

	// Prepare the message to be sent to the DA layer
	msg := interchainda.MsgSubmitBlob{
		Creator: c.accountAddress,
		Blob:    blob,
		Fees:    feesToPay,
	}

	// Broadcast the message to the DA layer applying retries in case of failure
	var txResp cosmosclient.Response
	err = c.runWithRetry(func() error {
		txResp, err = c.broadcastTx(&msg)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("can't broadcast MsgSubmitBlob to DA layer: %w", err)
	}

	// Wait until the tx in included into the DA layer
	rawResp, err := c.waitResponse(txResp.TxHash)
	if err != nil {
		return nil, fmt.Errorf("can't check acceptance of the blob to DA layer: %w", err)
	}
	if rawResp.TxResponse.Code != 0 {
		return nil, fmt.Errorf("MsgSubmitBlob is not executed in DA layer (code %d): %s", rawResp.TxResponse.Code, rawResp.TxResponse.RawLog)
	}

	// cosmosclient.Response has convenient Decode method, so we reuse txResp to reuse it
	var resp interchainda.MsgSubmitBlobResponse
	txResp.TxResponse = rawResp.TxResponse
	err = txResp.Decode(&resp)
	if err != nil {
		return nil, fmt.Errorf("can't decode MsgSubmitBlob response: %w", err)
	}

	// Get Merkle proof of the blob ID inclusion
	key, err := collections.EncodeKeyWithPrefix(
		interchainda.BlobMetadataPrefix(),
		collcodec.NewUint64Key[interchainda.BlobID](),
		interchainda.BlobID(resp.BlobId),
	)
	if err != nil {
		return nil, fmt.Errorf("can't encode DA lakey store key: %w", err)
	}
	abciPath := fmt.Sprintf("/store/%s/key", interchainda.StoreKey)
	abciResp, err := c.daClient.ABCIQueryWithProof(c.ctx, abciPath, key, txResp.Height)
	if err != nil {
		return nil, fmt.Errorf("can't call ABCI query with proof for the BlobID %d: %w", resp.BlobId, err)
	}
	if abciResp.Response.IsErr() {
		return nil, fmt.Errorf("can't call ABCI query with proof for blob ID %d (code %d): %s",
			resp.BlobId, abciResp.Response.Code, abciResp.Response.Log)
	}
	if abciResp.Response.Value == nil {
		return nil, fmt.Errorf("ABCI query with proof for blob ID %d returned nil value", resp.BlobId)
	}

	return &interchainda.Commitment{
		ClientId:    c.daConfig.ClientID,
		BlobHeight:  uint64(txResp.Height),
		BlobHash:    resp.BlobHash,
		BlobId:      resp.BlobId,
		MerkleProof: abciResp.Response.ProofOps,
	}, nil
}

func (c *DALayerClient) broadcastTx(msgs ...sdk.Msg) (cosmosclient.Response, error) {
	txResp, err := c.daClient.BroadcastTx(c.daConfig.AccountName, msgs...)
	if err != nil {
		return cosmosclient.Response{}, fmt.Errorf("can't broadcast MsgSubmitBlob to the DA layer: %w", err)
	}
	if txResp.Code != 0 {
		return cosmosclient.Response{}, fmt.Errorf("MsgSubmitBlob broadcast tx status code is not 0 (code %d): %s", txResp.Code, txResp.RawLog)
	}
	return txResp, nil
}

// runWithRetry runs the given operation with retry, doing a number of attempts, and taking the last error only.
func (c *DALayerClient) runWithRetry(operation func() error) error {
	return retry.Do(
		operation,
		retry.Context(c.ctx),
		retry.LastErrorOnly(true),
		retry.Delay(c.daConfig.RetryMinDelay),
		retry.Attempts(c.daConfig.RetryAttempts),
		retry.MaxDelay(c.daConfig.RetryMaxDelay),
		retry.DelayType(retry.BackOffDelay),
	)
}

func (c *DALayerClient) waitResponse(txHash string) (*tx.GetTxResponse, error) {
	timer := time.NewTicker(c.daConfig.BatchAcceptanceTimeout)
	defer timer.Stop()

	var txResp *tx.GetTxResponse
	attempt := uint(0)

	// First try then wait for the BatchAcceptanceTimeout
	for {
		err := c.runWithRetry(func() error {
			var errX error
			txResp, errX = c.daClient.GetTx(c.ctx, txHash)
			return errX
		})
		if err == nil {
			return txResp, nil
		}

		c.logger.Error("Can't check batch acceptance",
			"attempt", attempt, "max_attempts", c.daConfig.BatchAcceptanceAttempts, "error", err)

		attempt++
		if attempt > c.daConfig.BatchAcceptanceAttempts {
			return nil, fmt.Errorf("can't check batch acceptance after all attempts")
		}

		// Wait for the timeout
		select {
		case <-c.ctx.Done():
			return nil, c.ctx.Err()

		case <-timer.C:
			continue
		}
	}
}
