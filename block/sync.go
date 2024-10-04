package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewStateUpdate will try to sync to new height, if not already synced
func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatchAccepted)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}
	m.LastSubmittedHeight.Store(eventData.EndHeight)
	m.UpdateTargetHeight(eventData.EndHeight)
	select {
	case m.syncingC <- eventData.EndHeight:
	default:
		m.logger.Debug("disregarding new state update, since node is still syncing")
	}
}

// SyncTargetLoop is responsible for getting real time updates about settlement batch submissions.
// For non aggregator: updating the sync target which will be used by retrieveLoop to sync until this target.
// It publishes new sync height targets which will then be synced by another process.
func (m *Manager) SyncTargetLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case targetHeight := <-m.syncingC:
			err := m.UpdateSequencerSetFromSL()
			if err != nil {
				return fmt.Errorf("update bonded sequencer set: %w", err)
			}

			err = m.syncToLastSubmittedHeight()
			if err != nil {
				return fmt.Errorf("syncing to target height: %d err:%w", targetHeight, err)
			}
			m.synced.Nudge()
			m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSubmittedHeight.Load())
		}
	}
}

func (m *Manager) waitForSyncing() {
	if m.State.Height() < m.LastSubmittedHeight.Load() {
		<-m.synced.C
	}
}
