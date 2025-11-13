package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/settlement"
)

// onNewStateUpdate will update the last submitted height and will update sequencers list from SL. After, it triggers syncing or validation, depending whether it needs to sync first or only validate.
func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	// Update heights based on state update end height
	metrics.RollappHubHeightGauge.Set(float64(eventData.EndHeight))
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
			return nil
		case <-m.settlementSyncingC:
			m.logger.Info("syncing to target height", "targetHeight", m.LastSettlementHeight.Load())

			for currH := m.State.NextHeight(); currH <= m.LastSettlementHeight.Load(); currH = m.State.NextHeight() {
				// if context has been cancelled, stop syncing
				if ctx.Err() != nil {
					return nil
				}
				// if we have the block locally, we don't need to fetch it from the DA.
				// it will only happen in case of rollback.
				err := m.applyLocalBlock()
				if err == nil {
					m.logger.Info("Synced from local", "store height", m.State.Height(), "target height", m.LastSettlementHeight.Load())
					continue
				}
				if !errors.Is(err, gerrc.ErrNotFound) {
					m.logger.Error("Apply local block", "err", err)
				}

				settlementBatch, err := m.SLClient.GetBatchAtHeight(m.State.NextHeight())
				if err != nil {
					// TODO: should be recoverable. set to unhealthy and continue
					return fmt.Errorf("retrieve SL batch err: %w", err)
				}
				m.logger.Info("Retrieved state update from SL.", "state_index", settlementBatch.StateIndex)

				// we update LastSubmissionTime to be able to measure batch submission time
				m.LastSubmissionTime.Store(settlementBatch.Batch.CreationTime.UTC().UnixNano())

				err = m.ApplyBatchFromSL(settlementBatch.Batch)
				// this will keep sync loop alive when DA is down or retrievals are failing because DA issues.
				if errors.Is(err, da.ErrRetrieval) {
					// TODO: set to unhealthy?
					continue
				}
				if err != nil {
					return fmt.Errorf("process next DA batch. err:%w", err)
				}

				m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSettlementHeight.Load())

				// trigger state update validation, after each state update is applied
				m.triggerSettlementValidation()

			}

			// after syncing from SL, attempt to apply cached blocks if any
			err := m.attemptApplyCachedBlocks()
			if err != nil {
				return fmt.Errorf("attempt apply cached blocks: %w", err)
			}

			// avoid notifying as synced in case it fails before
			if m.State.Height() >= m.LastSettlementHeight.Load() {
				m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSettlementHeight.Load())
				// nudge to signal to any listens that we're currently synced with the last settlement height we've seen so far
				m.syncedFromSettlement.Nudge()
			}
		}
	}
}

// waitForSettlementSyncing waits for synced nudge (in case it needs to because it was syncing)
func (m *Manager) waitForSettlementSyncing() {
	if m.State.Height() < m.LastSettlementHeight.Load() {
		<-m.syncedFromSettlement.C
	}
}

// triggerSettlementSyncing sends signal to channel used by syncing loop
func (m *Manager) triggerSettlementSyncing() {
	select {
	case m.settlementSyncingC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

// triggerSettlementValidation sends signal to channel used by validation loop
func (m *Manager) triggerSettlementValidation() {
	select {
	case m.settlementValidationC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still validating")
	}
}
