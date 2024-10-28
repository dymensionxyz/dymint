package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/node/events"
	"github.com/dymensionxyz/dymint/settlement"
	uevent "github.com/dymensionxyz/dymint/utils/event"
)

// onNewStateUpdate will update the last submitted height and will update sequencers list from SL. After, it triggers syncing or validation, depending whether it needs to sync first or only validate.
func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	// Update heights based on state update end height
	m.LastSettlementHeight.Store(eventData.EndHeight)

	// Update sequencers list from SL
	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		// this error is not critical
		m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
	}

	if eventData.EndHeight > m.State.Height() {
		// trigger syncing from settlement last state update.
		m.triggerSettlementSyncing()
		// update target height used for syncing status rpc
		m.UpdateTargetHeight(eventData.EndHeight)
	} else {
		// trigger validation of the last state update available in settlement
		m.triggerSettlementValidation()
	}
}

// SettlementSyncLoop listens for syncing triggers which indicate new settlement height updates, and attempts to sync to the last seen settlement height.
// Syncing triggers can be called when a new settlement state update event arrives or explicitly from the `updateFromLastSettlementState` method which is only being called upon startup.
// Upon new trigger, we know the settlement reached a new height we haven't seen before so a validation signal is sent to validate the settlement batch.

// Note: even when a sync is triggered, there is no guarantee that the batch will be applied from settlement as there is a race condition with the p2p/blocksync for syncing.
func (m *Manager) SettlementSyncLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-m.settlementSyncingC:

			m.logger.Info("syncing to target height", "targetHeight", m.LastSettlementHeight.Load())

			for currH := m.State.NextHeight(); currH <= m.LastSettlementHeight.Load(); currH = m.State.NextHeight() {
				// if we have the block locally, we don't need to fetch it from the DA.
				// it will only happen in case of rollback.
				err := m.applyLocalBlock(currH)
				if err == nil {
					m.logger.Info("Synced from local", "store height", currH, "target height", m.LastSettlementHeight.Load())
					continue
				}
				if !errors.Is(err, gerrc.ErrNotFound) {
					m.logger.Error("Apply local block", "err", err)
				}

				settlementBatch, err := m.SLClient.GetBatchAtHeight(m.State.NextHeight())
				if err != nil {
					return fmt.Errorf("retrieve batch: %w", err)
				}
				m.logger.Info("Retrieved state update from SL.", "state_index", settlementBatch.StateIndex)

				err = m.ApplyBatchFromSL(settlementBatch.MetaData.DA)
				if err != nil {
					m.logger.Error("process next DA batch", "err", err)
				}

				// if height havent been updated, we are stuck
				if m.State.NextHeight() == currH {
					return fmt.Errorf("stuck at height %d", currH)
				}

				m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSettlementHeight.Load())

				// trigger state update validation, after each state update is applied
				m.triggerSettlementValidation()

				err = m.attemptApplyCachedBlocks()
				if err != nil {
					uevent.MustPublish(context.TODO(), m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					m.logger.Error("Attempt apply cached blocks.", "err", err)
				}

			}

			m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSettlementHeight.Load())

			// nudge to signal to any listens that we're currently synced with the last settlement height we've seen so far
			m.syncedFromSettlement.Nudge()

		}
	}
}

// waitForSyncing waits for synced nudge (in case it needs to because it was syncing)
func (m *Manager) waitForSettlementSyncing() {
	if m.State.Height() < m.LastSettlementHeight.Load() {
		<-m.syncedFromSettlement.C
	}
}

// triggerStateUpdateSyncing sends signal to channel used by syncing loop
func (m *Manager) triggerSettlementSyncing() {
	select {
	case m.settlementSyncingC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

// triggerStateUpdateValidation sends signal to channel used by validation loop
func (m *Manager) triggerSettlementValidation() {
	select {
	case m.settlementValidationC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still validating")
	}
}
