package block

import (
	"context"
	"fmt"
)

// SyncTargetLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// In case the node is already synced, it validate
func (m *Manager) ValidateLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.validateC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSubmittedHeight.Load())

			for currH := m.State.NextValidationHeight(); currH <= m.LastSubmittedHeight.Load(); currH = m.State.NextValidationHeight() {

				// get next batch that needs to be validated from SL
				batch, err := m.SLClient.GetBatchAtHeight(currH)
				if err != nil {
					m.logger.Error("failed batch retrieval", "error", err)
					break
				}
				// validate batch
				err = m.validator.ValidateStateUpdate(batch)
				if err != nil {
					panic(err)
				}

				// this should not happen. if validation is successful m.State.NextValidationHeight() should advance.
				if currH == m.State.NextValidationHeight() {
					panic("validation not progressing")
				}

				// update state with new validation height
				_, err = m.Store.SaveState(m.State, nil)
				if err != nil {
					return fmt.Errorf("save state: %w", err)
				}

				m.logger.Debug("state info validated", "batch end height", batch.EndHeight, "lastValidatedHeight", m.State.GetLastValidatedHeight())
			}

		}
	}
}
