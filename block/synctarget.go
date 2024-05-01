package block

import (
	"context"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/settlement"
)

// SyncTargetLoop is responsible for getting real time updates about batches submission.
// for non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	subscription, err := m.Pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		m.logger.Error("subscribe to state update events", "error", err)
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Out():
			eventData := event.Data().(*settlement.EventDataNewBatchAccepted)
			if eventData.EndHeight <= m.Store.Height() {
				m.logger.Debug(
					"syncTargetLoop: received new settlement batch accepted with batch end height <= current store height, skipping.",
					"height",
					eventData.EndHeight,
					"currentHeight",
					m.Store.Height(),
				)
				continue
			}
			m.UpdateSyncParams(eventData.EndHeight)
			m.SyncTargetDiode.Set(diodes.GenericDataType(&eventData.EndHeight))
		case <-subscription.Cancelled():
			m.logger.Info("syncTargetLoop subscription canceled")
			return
		}
	}
}
