package block

import (
	"context"

	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/tendermint/tendermint/libs/pubsub"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/settlement"
)

// SyncTargetLoop is responsible for getting real time updates about batches submission.
// for non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	uevent.MustSubscribe(
		ctx,
		m.Pubsub,
		"syncTargetLoop",
		settlement.EventQueryNewSettlementBatchAccepted,
		func(event pubsub.Message) {
			eventData := event.Data().(*settlement.EventDataNewBatchAccepted)
			if eventData.EndHeight <= m.Store.Height() {
				m.logger.Debug(
					"syncTargetLoop: received new settlement batch accepted with batch end height <= current store height, skipping.",
					"height",
					eventData.EndHeight,
					"currentHeight",
					m.Store.Height(),
				)
				return
			}
			m.UpdateSyncParams(eventData.EndHeight)
			m.SyncTargetDiode.Set(diodes.GenericDataType(&eventData.EndHeight))
		},
		m.logger,
	)
}
