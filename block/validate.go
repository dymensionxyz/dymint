package block

import "context"

// SyncTargetLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// In case the node is already synced, it validate
func (m *Manager) ValidateLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.validateC:

			m.logger.Info("validating state updates to target height", "targetHeight", m.LastSubmittedHeight.Load())

			validationStartHeight := max(m.LastSubmittedHeight.Load(), m.LastFinalizedHeight.Load()) + 1
			for currH := validationStartHeight; currH <= m.State.Height(); currH = m.LastSubmittedHeight.Load() {

				// Validate new state update
				batch, err := m.SLClient.GetBatchAtHeight(m.LastSubmittedHeight.Load())

				if err != nil {
					return err
				}
				err = m.validator.ValidateStateUpdate(batch)
				if err != nil {
					m.logger.Error("state update validation", "error", err)
				}
			}
		}
	}
}
