package stub

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

var _ da.DataAvailabilityLayerClient = &Layer{}

type Layer struct {
}

func (l Layer) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	//TODO implement me
	panic("implement me")
}

func (l Layer) DAPath() string {
	return ""
}

func (l Layer) Start() error {
	//TODO implement me
	panic("implement me")
}

func (l Layer) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (l Layer) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	//TODO implement me
	panic("implement me")
}

func (l Layer) GetClientType() da.Client {
	//TODO implement me
	panic("implement me")
}

func (l Layer) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	//TODO implement me
	panic("implement me")
}

func (l Layer) WaitForSyncing() {
	//TODO implement me
	panic("implement me")
}

func (l Layer) GetMaxBlobSizeBytes() uint32 {
	//TODO implement me
	panic("implement me")
}

func (l Layer) GetSignerBalance() (da.Balance, error) {
	//TODO implement me
	panic("implement me")
}
