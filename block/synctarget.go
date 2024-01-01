package block

import (
	"context"
	"sync/atomic"
	"time"

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
// for aggregator: get notification that batch has been accepted so can send next batch.
func (m *Manager) SyncTargetLoop(ctx context.Context) {
	m.logger.Info("Started sync target loop")
	subscription, err := m.pubsub.Subscribe(ctx, "syncTargetLoop", settlement.EventQueryNewSettlementBatchAccepted)
	if err != nil {
		m.logger.Error("failed to subscribe to state update events")
		panic(err)
	}
	// First time we start we want to get the latest batch from the SL
	resultRetrieveBatch, err := m.getLatestBatchFromSL(ctx)
	if err != nil {
		//FIXME: no better way to check if no batches yet or there's an error?
		m.logger.Error("failed to retrieve batch from SL", "err", err)
	} else {
		m.updateSyncParams(ctx, resultRetrieveBatch.EndHeight)
	}
	for {
		select {
		case <-ctx.Done():
			return
		//FIXME: add polling timer in case we missed an event
		case event := <-subscription.Out():
			eventData := event.Data().(*settlement.EventDataNewSettlementBatchAccepted)
			m.updateSyncParams(ctx, eventData.EndHeight)
			// In case we are the aggregator and we've got an update, then we can stop blocking from
			// the next batches to be published. For non-aggregators this is not needed.
			// We only want to send the next once the previous has been published successfully.
			// TODO(omritoptix): Once we have leader election, we can add a condition.
			// Update batch accepted is only relevant for the aggregator
			// TODO(omritoptix): Check if we are the aggregator
			m.batchInProcess.Store(false)
		case <-subscription.Cancelled():
			m.logger.Info("syncTargetLoop subscription canceled")
			return
		}
	}
}

// updateSyncParams updates the sync target and state index if necessary
func (m *Manager) updateSyncParams(ctx context.Context, endHeight uint64) {
	rollappHubHeightGauge.Set(float64(endHeight))
	m.logger.Info("Received new syncTarget", "syncTarget", endHeight)
	atomic.StoreUint64(&m.syncTarget, endHeight)
	atomic.StoreInt64(&m.lastSubmissionTime, time.Now().UnixNano())
	m.syncTargetDiode.Set(diodes.GenericDataType(&endHeight))
}
