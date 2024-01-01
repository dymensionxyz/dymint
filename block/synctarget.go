package block

import (
	"context"

	"code.cloudfoundry.org/go-diodes"
	"github.com/dymensionxyz/dymint/settlement"
)

// FLOW:
//(settlement/base.go#L160)
// hubclient get event
// hub client sends internal event

// synctartloop gets internal event
// synctartloop updates sync target
// retrieve loop gets sync target
// retrieve loop retrieves batch

// SyncTargetLoop is responsible for getting real time updates about batches submission.
// for non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	subscription, err := m.pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		m.logger.Error("failed to subscribe to state update events")
		panic(err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		//FIXME: add polling timer in case we missed an event
		case event := <-subscription.Out():
			eventData := event.Data().(*settlement.EventDataNewSettlementBatchAccepted)

			//FIXME: make sure the event is not old	/ replay

			m.updateSyncParams(ctx, eventData.EndHeight)
			m.syncTargetDiode.Set(diodes.GenericDataType(&eventData.EndHeight))
		case <-subscription.Cancelled():
			m.logger.Info("syncTargetLoop subscription canceled")
			return
		}
	}
}
