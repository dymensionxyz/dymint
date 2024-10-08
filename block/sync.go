package block

import (
	"context"
	"errors"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewStateUpdate will update the last submitted height and will update sequencers list from SL
func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatchAccepted)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	// Update heights based on state update end height
	m.LastSubmittedHeight.Store(eventData.EndHeight)
	m.UpdateTargetHeight(eventData.EndHeight)

	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		m.logger.Error("update bonded sequencer set", "error", err)
	}

	// Trigger syncing from DA.
	m.triggerStateUpdateSyncing()

}

// SyncTargetLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// In case the node is already synced, it validate
func (m *Manager) SyncLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.syncingC:

			m.logger.Info("syncing to target height", "targetHeight", m.LastSubmittedHeight.Load())

			for currH := m.State.NextHeight(); currH <= m.LastSubmittedHeight.Load(); currH = m.State.NextHeight() {
				// if we have the block locally, we don't need to fetch it from the DA
				err := m.applyLocalBlock(currH)
				if err == nil {
					m.logger.Info("Synced from local", "store height", currH, "target height", m.LastSubmittedHeight.Load())
					continue
				}
				if !errors.Is(err, gerrc.ErrNotFound) {
					m.logger.Error("Apply local block", "err", err)
				}

				err = m.syncFromDABatch()
				if err != nil {
					m.logger.Error("process next DA batch", "err", err)
				}

				m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSubmittedHeight.Load())

				m.triggerStateUpdateValidation()

				err = m.attemptApplyCachedBlocks()
				if err != nil {
					uevent.MustPublish(context.TODO(), m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					m.logger.Error("Attempt apply cached blocks.", "err", err)
				}

			}

			m.triggerStateUpdateValidation()

			m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSubmittedHeight.Load())

			m.synced.Nudge()

		}
	}
}

// waitForSyncing waits for synced nudge (in case it needs to because it was syncing)
func (m *Manager) waitForSyncing() {
	if m.State.Height() < m.LastSubmittedHeight.Load() {
		<-m.synced.C
	}
}

func (m *Manager) triggerStateUpdateSyncing() {
	// Trigger state update validation.
	select {
	case m.syncingC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

func (m *Manager) triggerStateUpdateValidation() {
	// Trigger state update validation.
	select {
	case m.validateC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still validating")
	}
}
