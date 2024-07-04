package interchain

import (
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	interchainda "github.com/dymensionxyz/dymint/types/pb/interchain_da"
)

func (c *DALayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	panic("RetrieveBatches method is not supported by the interchain DA clint")
}

func (c *DALayerClient) RetrieveBatchesV2(b da.ResultSubmitBatchV2) da.ResultRetrieveBatchV2 {
	batch, err := c.retrieveBatches(b)
	if err != nil {
		return da.ResultRetrieveBatchV2{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("can't retrieve batch from the interchain DA layer: %s", err.Error()),
				Error:   err,
			},
			Batch: types.Batch{},
		}
	}

	return da.ResultRetrieveBatchV2{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Retrieve successful",
		},
		Batch: batch,
	}
}

func (c *DALayerClient) retrieveBatches(b da.ResultSubmitBatchV2) (types.Batch, error) {
	var commitment *interchainda.Commitment
	err := c.cdc.UnpackAny(b.DAPath.Commitment, &commitment)
	if err != nil {
		return types.Batch{}, fmt.Errorf("can't unpack commitment: %w", err)
	}

	resp, err := c.daClient.Blob(c.ctx, interchainda.BlobID(commitment.BlobId))
	if err != nil {
		return types.Batch{}, fmt.Errorf("can't get blob from the interchain DA layer: %w", err)
	}

	if resp.BlobMetadata.BlobHash != commitment.BlobHash {
		return types.Batch{}, fmt.Errorf("commitment blob hash doesn't match interchain DA layer blob hash")
	}

	batch, err := DecodeBatch(resp.Blob)
	if err != nil {
		return types.Batch{}, fmt.Errorf("can't decode batch from interchain DA layer: %w", err)
	}

	return batch, nil
}
