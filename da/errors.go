package da

import (
	"errors"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

var (
	ErrTxBroadcastConfigError = errors.New("failed building tx")

	ErrTxBroadcastNetworkError = errors.New("failed broadcasting tx")

	ErrTxBroadcastTimeout = errors.New("broadcast timeout error")

	ErrUnableToGetProof = errors.New("unable to get proof")

	ErrRetrieval = errors.New("retrieval failed")

	ErrBlobNotFound = errors.New("blob not found")

	ErrBlobNotIncluded = errors.New("blob not included")

	ErrBlobNotParsed = errors.New("unable to parse blob to batch")

	ErrProofNotMatching = errors.New("proof not matching")

	ErrNameSpace = errors.New("namespace not matching")

	ErrDAMismatch = gerrc.ErrInvalidArgument.Wrap("DA in config not matching DA path")
)
