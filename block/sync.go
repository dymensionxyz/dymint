package block

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"

	"github.com/dymensionxyz/dymint/settlement"
)

func (m *Manager) onNewStateUpdate(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}

	m.LastSettlementHeight.Store(eventData.EndHeight)

	err := m.UpdateSequencerSetFromSL()
	if err != nil {
		m.logger.Error("Cannot fetch sequencer set from the Hub", "error", err)
	}

	if eventData.EndHeight > m.State.Height() {

		m.triggerSettlementSyncing()

		m.UpdateTargetHeight(eventData.EndHeight)
	} else {
		m.triggerSettlementValidation()
	}
}

func (m *Manager) SettlementSyncLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.settlementSyncingC:
			m.logger.Info("syncing to target height", "targetHeight", m.LastSettlementHeight.Load())

			for currH := m.State.NextHeight(); currH <= m.LastSettlementHeight.Load(); currH = m.State.NextHeight() {

				if ctx.Err() != nil {
					return nil
				}

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
					return fmt.Errorf("retrieve SL batch err: %w", err)
				}
				m.logger.Info("Retrieved state update from SL.", "state_index", settlementBatch.StateIndex)

				m.LastBlockTimeInSettlement.Store(settlementBatch.BlockDescriptors[len(settlementBatch.BlockDescriptors)-1].GetTimestamp().UTC().UnixNano())

				err = m.ApplyBatchFromSL(settlementBatch.Batch)

				if errors.Is(err, da.ErrRetrieval) {
					continue
				}
				if err != nil {
					return fmt.Errorf("process next DA batch. err:%w", err)
				}

				m.logger.Info("Synced from DA", "store height", m.State.Height(), "target height", m.LastSettlementHeight.Load())

				m.triggerSettlementValidation()

				err = m.attemptApplyCachedBlocks()
				if err != nil {
					return fmt.Errorf("Attempt apply cached blocks. err:%w", err)
				}

			}

			if m.State.Height() >= m.LastSettlementHeight.Load() {
				m.logger.Info("Synced.", "current height", m.State.Height(), "last submitted height", m.LastSettlementHeight.Load())

				m.syncedFromSettlement.Nudge()
			}

		}
	}
}

func (m *Manager) waitForSettlementSyncing() {
	if m.State.Height() < m.LastSettlementHeight.Load() {
		<-m.syncedFromSettlement.C
	}
}

func (m *Manager) triggerSettlementSyncing() {
	select {
	case m.settlementSyncingC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still syncing")
	}
}

func (m *Manager) triggerSettlementValidation() {
	select {
	case m.settlementValidationC <- struct{}{}:
	default:
		m.logger.Debug("disregarding new state update, node is still validating")
	}
}
