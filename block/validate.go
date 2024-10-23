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

// onNewStateUpdateFinalized will update the last validated height with the last finalized height.
// Unlike pending heights, once heights are finalized, we treat them as validated as there is no point validating finalized heights.
func (m *Manager) onNewStateUpdateFinalized(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onNewStateUpdateFinalized", "err", "wrong event data received")
		return
	}
	m.UpdateLastValidatedHeight(eventData.EndHeight)
}

// ValidateLoop listens for syncing events (from new state update or from initial syncing) and validates state updates to the last submitted height.
func (m *Manager) ValidateLoop(ctx context.Context) error {
	lastValidatedHeight, err := m.Store.LoadValidationHeight()
	if err != nil {
		m.logger.Debug("validation height not loaded", "err", err)
	}
	m.UpdateLastValidatedHeight(lastValidatedHeight)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.settlementValidationC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSettlementHeight.Load())

			for currH := m.NextValidationHeight(); currH <= m.LastSettlementHeight.Load(); currH = m.NextValidationHeight() {

				// get next batch that needs to be validated from SL
				batch, err := m.SLClient.GetBatchAtHeight(currH)
				if err != nil {
					uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					return err
				}
				// validate batch
				err = m.settlementValidator.ValidateStateUpdate(batch)
				if err != nil {
					if errors.Is(err, gerrc.ErrFault) {
						m.FraudHandler.HandleFault(ctx, err)
					} else {
						uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					}
					return err
				}

				// update the last validated height to the batch last block height
				m.UpdateLastValidatedHeight(batch.EndHeight)

				m.logger.Debug("state info validated", "batch end height", batch.EndHeight, "lastValidatedHeight", m.GetLastValidatedHeight())
			}

		}
	}
}

// UpdateLastValidatedHeight sets the height saved in the Store if it is higher than the existing height
// returns OK if the value was updated successfully or did not need to be updated
func (m *Manager) UpdateLastValidatedHeight(height uint64) {
	for {
		curr := m.lastValidatedHeight.Load()
		if m.lastValidatedHeight.CompareAndSwap(curr, max(curr, height)) {
			_, err := m.Store.SaveValidationHeight(m.GetLastValidatedHeight(), nil)
			if err != nil {
				m.logger.Error("update validation height: %w", err)
			}
			break
		}
	}
}

// GetLastValidatedHeight returns the most last block height that is validated with settlement state updates.
func (m *Manager) GetLastValidatedHeight() uint64 {
	return m.lastValidatedHeight.Load()
}

// GetLastValidatedHeight returns the next height that needs to be validated with settlement state updates.
func (m *Manager) NextValidationHeight() uint64 {
	return m.lastValidatedHeight.Load() + 1
}
