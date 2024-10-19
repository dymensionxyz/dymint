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
		m.logger.Error("onReceivedBatch", "err", "wrong event data received")
		return
	}
	m.State.UpdateLastValidatedHeight(eventData.EndHeight)
}

// ValidateLoop listens for syncing events (from new state update or from initial syncing) and validates state updates to the last submitted height.
func (m *Manager) ValidateLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.validateC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSubmittedHeight.Load())

			for currH := m.State.NextValidationHeight(); currH <= m.LastSubmittedHeight.Load(); currH = m.State.NextValidationHeight() {

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
					panic(err)
				}

				// this should not happen. if validation is successful m.State.NextValidationHeight() should advance.
				if currH == m.State.NextValidationHeight() {
					panic("validation not progressing")
				}

				// update state with new validation height
				_, err = m.Store.SaveState(m.State, nil)
				if err != nil {
					m.logger.Error("save state: %w", err)
				}

				m.logger.Debug("state info validated", "batch end height", batch.EndHeight, "lastValidatedHeight", m.State.GetLastValidatedHeight())
			}

		}
	}
}
