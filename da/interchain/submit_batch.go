package interchain

import (
	"fmt"

	"cosmossdk.io/collections"
	collcodec "cosmossdk.io/collections/codec"
	"github.com/avast/retry-go/v4"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dymensionxyz/cosmosclient/cosmosclient"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/interchain/ioutils"
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
	// Prepare the blob data
	blob, err := batch.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("can't marshal batch: %w", err)
	}

	// Gzip the blob
	gzipped, err := ioutils.Gzip(blob)
	if err != nil {
		return nil, fmt.Errorf("can't gzip batch: %w", err)
	}

	// Verify the size of the blob is within the limit
	if len(blob) > int(c.daConfig.DAParams.MaxBlobSize) {
		return nil, fmt.Errorf("blob size %d exceeds the maximum allowed size %d", len(blob), c.daConfig.DAParams.MaxBlobSize)
	}

	// Calculate the fees of submitting this blob
	feesToPay := sdk.NewCoin(c.daConfig.DAParams.CostPerByte.Denom, c.daConfig.DAParams.CostPerByte.Amount.MulRaw(int64(len(blob))))

	// Prepare the message to be sent to the DA layer
	msg := interchainda.MsgSubmitBlob{
		Creator: c.daConfig.AccountName,
		Blob:    gzipped,
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

	// Decode the response
	var resp interchainda.MsgSubmitBlobResponse
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
	const keyPath = "/key"
	abciResp, err := c.daClient.ABCIQueryWithProof(c.ctx, keyPath, key, txResp.Height)
	if err != nil {
		return nil, fmt.Errorf("can't call ABCI query with proof for the BlobID %d: %w", resp.BlobId, err)
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
		return cosmosclient.Response{}, fmt.Errorf("MsgSubmitBlob broadcast tx status code is not 0: code %d", txResp.Code)
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
