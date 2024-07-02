package interchain

import "github.com/dymensionxyz/dymint/da"

func (c *DALayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	panic("RetrieveBatches method is not supported by the interchain DA clint")
}

func (c *DALayerClient) RetrieveBatchesV2(da.ResultSubmitBatchV2) da.ResultRetrieveBatchV2 {
	panic("implement me")
}
