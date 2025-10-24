package block

import (
	"errors"
	"fmt"

	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"

	rollapptypes "github.com/dymensionxyz/dymint/types/pb/dymensionxyz/dymension/rollapp"
)

func (m *Manager) RunInitChain() error {
	// Get the proposer at the initial height. If we're at genesis the height will be 0.
	proposer, err := m.SLClient.GetProposerAtHeight(int64(m.State.Height()) + 1) //nolint:gosec // height is non-negative and falls in int64
	if err != nil {
		return fmt.Errorf("get proposer at height: %w", err)
	}
	if proposer == nil {
		return errors.New("failed to get proposer")
	}
	tmProposer := proposer.TMValidator()
	res, err := m.Executor.InitChain(m.Genesis, m.GenesisChecksum, []*tmtypes.Validator{tmProposer})
	if err != nil {
		return err
	}

	// validate the resulting genesis bridge data against the hub
	err = m.ValidateGenesisBridgeData(res.GenesisBridgeDataBytes)
	if err != nil {
		return fmt.Errorf("cannot validate genesis bridge data: %w. Please call `$EXECUTABLE dymint unsafe-reset-all` before the next launch to reset this node to genesis state", err)
	}

	// update the state with only the consensus pubkey
	m.Executor.UpdateStateAfterInitChain(m.State, res)
	m.Executor.UpdateMempoolAfterInitChain(m.State)
	if _, err := m.Store.SaveState(m.State, nil); err != nil {
		return err
	}

	return nil
}

// ValidateGenesisBridgeData validates the genesis bridge data from
// InitChainResponse against the rollapp genesis stored in the hub.
func (m *Manager) ValidateGenesisBridgeData(dataBytes []byte) error {
	if len(dataBytes) == 0 {
		return fmt.Errorf("genesis bridge data is empty in InitChainResponse")
	}
	var genesisBridgeData rollapptypes.GenesisBridgeData
	err := tmjson.Unmarshal(dataBytes, &genesisBridgeData)
	if err != nil {
		return fmt.Errorf("unmarshal genesis bridge data: %w", err)
	}
	return m.SLClient.ValidateGenesisBridgeData(genesisBridgeData)
}
