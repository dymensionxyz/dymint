package types

import (
	"encoding/json"
	"fmt"

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	abci "github.com/tendermint/tendermint/abci/types"
)

type RollappParams struct {
	Params *dymint.RollappParams
}

func RollappParamsToABCI(r dymint.RollappParams) abci.RollappParams {
	ret := abci.RollappParams{}
	ret.Da = r.Da
	ret.DrsVersion = r.DrsVersion
	return ret
}

// GetRollappParamsFromGenesis returns the rollapp consensus params from genesis
func GetRollappParamsFromGenesis(appState json.RawMessage) (RollappParams, error) {
	var objmap map[string]json.RawMessage
	err := json.Unmarshal(appState, &objmap)
	if err != nil {
		return RollappParams{}, err
	}
	params, ok := objmap[rollappparams_modulename]
	if !ok {
		return RollappParams{}, fmt.Errorf("module not defined in genesis: %s", rollappparams_modulename)
	}

	var rollappParams RollappParams
	err = json.Unmarshal(params, &rollappParams)
	if err != nil {
		return RollappParams{}, err
	}

	return rollappParams, nil
}
