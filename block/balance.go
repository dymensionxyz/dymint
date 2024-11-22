package block

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"strconv"
	"sync"
	"time"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/types"
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
				if amountFloat, err := strconv.ParseFloat(balances.DA.Amount.String(), 64); err == nil {
					types.DaLayerBalanceGauge.Set(amountFloat)
				} else {
					m.logger.Error("Parsing DA balance amount", "error", err)
				}
			}

			if balances.SL != nil {
				if amountFloat, err := strconv.ParseFloat(balances.SL.Amount.String(), 64); err == nil {
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
		balances.DA = balance
	}()

	go func() {
		defer wg.Done()
		balance, err := m.SLClient.GetSignerBalance()
		if err != nil {
			errSL = fmt.Errorf("get SL signer balance: %w", err)
			return
		}
		balances.SL = balance
	}()

	wg.Wait()

	var errs error
	if errDA != nil {
		errs = errors.Join(errs, errDA)
	}
	if errSL != nil {
		errs = errors.Join(errs, errSL)
	}

	if errs != nil {
		return balances, fmt.Errorf("errors checking balances: %w", errs)
	}

	return balances, nil
}
