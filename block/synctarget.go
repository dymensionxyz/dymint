package block

import (
	"context"

	"github.com/dymensionxyz/dymint/types"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/settlement"
)

// SyncToTargetHeightLoop is responsible for getting real time updates about settlement batch submissions.
// For non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
// It publishes new sync height targets which will then be synced by another process.
func (m *Manager) SyncToTargetHeightLoop(ctx context.Context) {
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
			h := eventData.EndHeight

			if h <= m.Store.Height() {
				m.logger.Debug(
					"syncTargetLoop: received new settlement batch accepted with batch end height <= current store height, skipping.",
					"target sync height (batch end height)",
					h,
					"current store height",
					m.Store.Height(),
				)
				continue
			}
			m.targetSyncHeight.Set(diodes.GenericDataType(&h))
			m.logger.Info("Set new target sync height", "height", h)
			types.RollappHubHeightGauge.Set(float64(h)) // TODO(danwt): needed?
		case <-subscription.Cancelled():
			m.logger.Info("syncTargetLoop subscription canceled")
			return
		}
	}
}
