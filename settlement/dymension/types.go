package dymension

import (
	"errors"

	dymension "github.com/dymensionxyz/dymension/v3/x/rollapp/types"
)

var (
	// if this error is returned, the batch is already submitted
	ErrBatchAlreadySubmitted = errors.New(dymension.ErrWrongBlockHeight.Error())
)
