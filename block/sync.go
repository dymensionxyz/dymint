package block

import (
	"context"
	"fmt"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewStateUpdate will update the last submitted height and will
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
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

// SyncTargetLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// In case the node is already synced, it validate
func (m *Manager) SyncTargetLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case targetHeight := <-m.syncingC:
			err := m.UpdateSequencerSetFromSL()
			if err != nil {
				m.logger.Error("update bonded sequencer set", "error", err)
			}
			if m.State.Height() < targetHeight {
				err = m.syncToLastSubmittedHeight()
				if err != nil {
					m.logger.Error("syncing to target height", "targetHeight", targetHeight, "error", err)
				}
				m.synced.Nudge()
				m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSubmittedHeight.Load())
			} else {
				err = m.validateToLastSubmittedHeight(targetHeight)
				if err != nil {
					return fmt.Errorf("syncing to target height: %d err:%w", targetHeight, err)
				}
			}
		}
	}
}

// waitForSyncing waits for synced nudge (in case it needs to because it was syncing)
func (m *Manager) waitForSyncing() {
	if m.State.Height() < m.LastSubmittedHeight.Load() {
		<-m.synced.C
	}
}
