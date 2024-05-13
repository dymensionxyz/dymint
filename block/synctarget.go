package block

import (
	"context"

	"github.com/dymensionxyz/dymint/types"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/settlement"
)

// SyncTargetLoop is responsible for getting real time updates about settlement batch submissions.
// For non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
// It publishes new sync height targets which will then be synced by another process.
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
			m.SyncTargetDiode.Set(diodes.GenericDataType(&eventData.EndHeight))
			m.logger.Info("Received new sync target height", "height", eventData.EndHeight)
			types.RollappHubHeightGauge.Set(float64(eventData.EndHeight)) // TODO(danwt): needed?
		case <-subscription.Cancelled():
			m.logger.Info("syncTargetLoop subscription canceled")
			return
		}
	}
}
