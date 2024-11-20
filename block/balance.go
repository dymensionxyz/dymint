package block

import (
	"context"
	"github.com/dymensionxyz/dymint/types"
	"strconv"
	"time"
)

// MonitorBalances checks the balances of the node and updates the gauges for prometheus
func (m *Manager) MonitorBalances(ctx context.Context) error {
	ticker := time.NewTicker(m.Conf.BlockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			m.logger.Info("Checking balances.")
			balances, err := m.checkBalances()

			if balances.DA != nil {
				if amountFloat, err := strconv.ParseFloat(balances.DA.Amount.String(), 64); err == nil {
					m.logger.Info("Setting DA balance gauge.", "amount", amountFloat)
					types.DaLayerBalanceGauge.Set(amountFloat)
				} else {
					m.logger.Error("Parsing DA balance amount", "error", err)
				}
			}

			if balances.SL != nil {
				if amountFloat, err := strconv.ParseFloat(balances.SL.Amount.String(), 64); err == nil {
					m.logger.Info("Setting SL balance gauge.", "amount", amountFloat)
					types.HubLayerBalanceGauge.Set(amountFloat)
				} else {
					m.logger.Error("Parsing SL balance amount", "error", err)
				}
			}

			if err != nil {
				m.logger.Error("Checking balances", "error", err)
			}
		}
	}
}
