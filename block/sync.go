package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	uevent "github.com/dymensionxyz/dymint/utils/event"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewStateUpdate will update the last submitted height and will update sequencers list from SL. After, it triggers syncing or validation, depending whether it needs to sync first or only validate.
func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	// Update heights based on state update end height
	m.LastSubmittedHeight.Store(eventData.EndHeight)
	m.UpdateTargetHeight(eventData.EndHeight)

	m.logger.Error("syncing")

	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		m.logger.Error("update bonded sequencer set", "error", err)
	}

	if eventData.EndHeight > m.State.Height() {
		// Trigger syncing from DA.
		m.triggerStateUpdateSyncing()
	} else {
		// trigger state update validation (in case no state update is applied)
		m.triggerStateUpdateValidation()
	}
}

// SyncLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// It sends signal to validation loop for each synced state updated
func (m *Manager) SyncLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

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

				// if height havent been updated, we are stuck
				if m.State.NextHeight() == currH {
					return fmt.Errorf("stuck at height %d", currH)
				}

				m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSubmittedHeight.Load())

				// trigger state update validation, after each state update is applied
				m.triggerStateUpdateValidation()

				err = m.attemptApplyCachedBlocks()
				if err != nil {
					uevent.MustPublish(context.TODO(), m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					m.logger.Error("Attempt apply cached blocks.", "err", err)
				}

			}

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

// triggerStateUpdateSyncing sends signal to channel used by syncing loop
func (m *Manager) triggerStateUpdateSyncing() {
	select {
	case m.syncingC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

// triggerStateUpdateValidation sends signal to channel used by validation loop
func (m *Manager) triggerStateUpdateValidation() {
	select {
	case m.validateC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still validating")
	}
}
