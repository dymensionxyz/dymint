package dymension

import (
	dymension "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
)

// if this error is returned, the batch is already submitted
var ErrBatchAlreadySubmitted = dymension.ErrWrongBlockHeight.Error()
