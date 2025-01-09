package types

import (
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
