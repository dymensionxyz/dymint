package block

import (
	"context"

	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/types"

	"code.cloudfoundry.org/go-diodes"

	"github.com/dymensionxyz/dymint/settlement"
)

// SyncToTargetHeightLoop gets real time updates about settlement batch submissions and sends the latest height downstream
// to be retrieved by another process which will pull the data.
func (m *Manager) SyncToTargetHeightLoop(ctx context.Context) (err error) {
	m.logger.Info("Started sync target loop")
	var subscription *pubsub.Subscription
	subscription, err = m.Pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		m.logger.Error("subscribe to state update events", "error", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-subscription.Out():
			eventData, _ := event.Data().(*settlement.EventDataNewBatchAccepted)
			h := eventData.EndHeight

			if h <= m.State.Height() {
				m.logger.Debug(
					"syncTargetLoop: received new settlement batch accepted with batch end height <= current store height, skipping.",
					"target sync height (batch end height)",
					h,
					"current store height",
					m.State.Height(),
				)
				continue
			}
			types.RollappHubHeightGauge.Set(float64(h))
			m.LastSubmittedHeight.Store(h)
			m.targetSyncHeight.Set(diodes.GenericDataType(&h))
			m.logger.Info("Set new target sync height", "height", h)
		case <-subscription.Cancelled():
			m.logger.Error("syncTargetLoop subscription canceled")
			return
		}
	}
}
