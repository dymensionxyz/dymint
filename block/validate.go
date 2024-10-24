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
	m.settlementValidator.UpdateLastValidatedHeight(eventData.EndHeight)
}

// ValidateLoop listens for syncing events (from new state update or from initial syncing) and validates state updates to the last submitted height.
func (m *Manager) ValidateLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-m.settlementValidationC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSettlementHeight.Load())

			for currH := m.settlementValidator.NextValidationHeight(); currH <= m.LastSettlementHeight.Load(); currH = m.settlementValidator.NextValidationHeight() {

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
				m.settlementValidator.UpdateLastValidatedHeight(batch.EndHeight)

				m.logger.Debug("state info validated", "batch end height", batch.EndHeight, "lastValidatedHeight", m.settlementValidator.GetLastValidatedHeight())
			}

		}
	}
}
