package da

import (
	"errors"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

var (
	// ErrFailedTxBuild is returned when transaction build fails.
	ErrTxBroadcastConfigError = errors.New("failed building tx")
	// ErrFailedTxBroadcast is returned when transaction broadcast fails.
	ErrTxBroadcastNetworkError = errors.New("failed broadcasting tx")
	// ErrTxBroadcastTimeout is returned when transaction broadcast times out.
	ErrTxBroadcastTimeout = errors.New("broadcast timeout error")

	// ErrRetrieval is returned when retrieval rpc falls
	ErrRetrieval = errors.New("retrieval failed")
	// ErrBlobNotFound is returned when blob is not found.
	ErrBlobNotFound = errors.New("blob not found")
	// ErrBlobNotIncluded is returned when blob is not included.
	ErrBlobNotIncluded = errors.New("blob not included")
	// ErrUnableToGetProofs is returned when proof is not available.
	ErrUnableToGetProofs = errors.Join(ErrBlobNotIncluded, errors.New("unable to get availability proofs"))
	// ErrProofNotMatching is returned when proof does not match.
	ErrProofNotMatching = errors.Join(ErrBlobNotIncluded, errors.New("proof not matching"))
	// ErrBlobNotParsed is returned when blob cannot be parsed
	ErrBlobNotParsed = errors.New("unable to parse blob to batch")

	// ErrNameSpace is returned when wrong namespace used
	ErrNameSpace = errors.New("namespace not matching")
	// ErrDAMismatch is returned when the DA client used does not match the da client specified in the da path of the state update
	ErrDAMismatch = gerrc.ErrInvalidArgument.Wrap("DA in config not matching DA path")
)
