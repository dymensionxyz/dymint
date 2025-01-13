package block

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
)

const CheckBalancesInterval = 3 * time.Minute

// MonitorBalances checks the balances of the node and updates the gauges for prometheus
func (m *Manager) MonitorBalances(ctx context.Context) error {
	ticker := time.NewTicker(CheckBalancesInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			m.logger.Info("Checking balances.")
			balances, err := m.checkBalances()

			if balances.DA != nil {
				if amountFloat, errDA := strconv.ParseFloat(balances.DA.Amount.String(), 64); errDA == nil {
					metrics.DaLayerBalanceGauge.Set(amountFloat)
				} else {
					m.logger.Error("Parsing DA balance amount", "error", errDA)
				}
			}

			if balances.SL != nil {
				if amountFloat, errSL := strconv.ParseFloat(balances.SL.Amount.String(), 64); errSL == nil {
					metrics.HubLayerBalanceGauge.Set(amountFloat)
				} else {
					m.logger.Error("Parsing SL balance amount", "error", errSL)
				}
			}

			if err != nil {
				m.logger.Error("Checking balances", "error", err)
			}
		}
	}
}

type Balances struct {
	DA *da.Balance
	SL *types.Balance
}

func (m *Manager) checkBalances() (*Balances, error) {
	balances := &Balances{}
	var wg sync.WaitGroup
	wg.Add(2)

	var errDA, errSL error

	go func() {
		defer wg.Done()
		balance, err := m.DAClient.GetSignerBalance()
		if err != nil {
			errDA = fmt.Errorf("get DA signer balance: %w", err)
			return
		}
		balances.DA = &balance
	}()

	go func() {
		defer wg.Done()
		balance, err := m.SLClient.GetSignerBalance()
		if err != nil {
			errSL = fmt.Errorf("get SL signer balance: %w", err)
			return
		}
		balances.SL = &balance
	}()

	wg.Wait()

	errs := errors.Join(errDA, errSL)
	if errs != nil {
		return balances, fmt.Errorf("errors checking balances: %w", errs)
	}

	return balances, nil
}
