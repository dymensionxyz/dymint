package block

import (
	"context"

	"github.com/dymensionxyz/dymint/settlement"
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
	m.SettlementValidator.UpdateLastValidatedHeight(eventData.EndHeight)
}

// SettlementValidateLoop listens for syncing events (from new state update or from initial syncing) and validates state updates to the last submitted height.
func (m *Manager) SettlementValidateLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.settlementValidationC:
			targetValidationHeight := min(m.LastSettlementHeight.Load(), m.State.Height())
			m.logger.Info("validating state updates to target height", "targetHeight", targetValidationHeight)

			for currH := m.SettlementValidator.NextValidationHeight(); currH <= targetValidationHeight; currH = m.SettlementValidator.NextValidationHeight() {
				// get next batch that needs to be validated from SL
				batch, err := m.SLClient.GetBatchAtHeight(currH, true)
				if err != nil {
					// TODO: should be recoverable. set to unhealthy and continue
					return err
				}

				// validate batch
				err = m.SettlementValidator.ValidateStateUpdate(batch)
				if err != nil {
					return err
				}

				// update the last validated height to the batch last block height
				m.SettlementValidator.UpdateLastValidatedHeight(batch.EndHeight)

				m.logger.Info("state info validated", "idx", batch.StateIndex, "start height", batch.StartHeight, "end height", batch.EndHeight)
			}
		}
	}
}
