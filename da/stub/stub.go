package stub

import (
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/tendermint/tendermint/libs/pubsub"
)

var _ da.DataAvailabilityLayerClient = &Layer{}

type Layer struct{}

func (l Layer) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	panic("implement me")
}

func (l Layer) DAPath() string {
	return ""
}

func (l Layer) Start() error {
	panic("implement me")
}

func (l Layer) Stop() error {
	panic("implement me")
}

func (l Layer) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	panic("implement me")
}

func (l Layer) GetClientType() da.Client {
	panic("implement me")
}

func (l Layer) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	panic("implement me")
}

func (l Layer) WaitForSyncing() {
	panic("implement me")
}

func (l Layer) GetMaxBlobSizeBytes() uint64 {
	panic("implement me")
}

func (l Layer) GetSignerBalance() (da.Balance, error) {
	panic("implement me")
}
