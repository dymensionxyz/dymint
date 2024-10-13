package block

import (
	"context"
	"errors"

	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/pubsub"
)

// onNewStateUpdateFinalized will update the last validated height with the last finalized height
func (m *Manager) onNewStateUpdateFinalized(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		m.logger.Error("onNewStateUpdateFinalized", "err", "wrong event data received")
		return
	}
	m.UpdateLastValidatedHeight(eventData.EndHeight)
	m.State.SetLastFinalizedHeight(eventData.EndHeight)

	// remove drs heights record for already finalized heights, since we do not need them for validation
	m.State.ClearDRSVersionHeights(eventData.EndHeight)
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
		case <-m.validateC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSubmittedHeight.Load())

			for currH := m.NextValidationHeight(); currH <= m.LastSubmittedHeight.Load(); currH = m.NextValidationHeight() {

				// get next batch that needs to be validated from SL
				batch, err := m.SLClient.GetBatchAtHeight(currH)
				if err != nil {
					m.logger.Error("failed batch retrieval", "error", err)
					continue
				}
				// validate batch
				err = m.validator.ValidateStateUpdate(batch)
				if errors.Is(err, gerrc.ErrFault) {
					m.FraudHandler.HandleFault(ctx, err)
				} else if err != nil {
					m.logger.Error("validate loop", "err", err)
				}

				// this should not happen. if validation is successful m.NextValidationHeight() should advance.
				if currH == m.NextValidationHeight() {
					panic("validation not progressing")
				}

				_, err = m.Store.SaveValidationHeight(m.GetLastValidatedHeight(), nil)
				if err != nil {
					m.logger.Error("update validation height: %w", err)
				}

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
			break
		}
	}
}

// Height returns height of the highest block saved in the Store.
func (m *Manager) GetLastValidatedHeight() uint64 {
	return m.lastValidatedHeight.Load()
}

// Height returns height of the highest block saved in the Store.
func (m *Manager) NextValidationHeight() uint64 {
	return m.lastValidatedHeight.Load() + 1
}
