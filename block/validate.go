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

func (m *Manager) onNewStateUpdateFinalized(event pubsub.Message) {
	eventData, ok := event.Data().(*settlement.EventDataNewBatch)
	if !ok {
		return
	}
	m.SettlementValidator.UpdateLastValidatedHeight(eventData.EndHeight)
}

func (m *Manager) SettlementValidateLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.settlementValidationC:
			targetValidationHeight := min(m.LastSettlementHeight.Load(), m.State.Height())

			for currH := m.SettlementValidator.NextValidationHeight(); currH <= targetValidationHeight; currH = m.SettlementValidator.NextValidationHeight() {

				batch, err := m.SLClient.GetBatchAtHeight(currH)
				if err != nil {
					uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					return err
				}

				err = m.SettlementValidator.ValidateStateUpdate(batch)
				if err != nil {
					if errors.Is(err, gerrc.ErrFault) {
						m.FraudHandler.HandleFault(ctx, err)
					} else {
						uevent.MustPublish(ctx, m.Pubsub, &events.DataHealthStatus{Error: err}, events.HealthStatusList)
					}
					return err
				}

				m.SettlementValidator.UpdateLastValidatedHeight(batch.EndHeight)

			}

		}
	}
}
