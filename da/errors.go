package da

import "errors"

var (
	// ErrFailedTxBuild is returned when transaction build fails.
	ErrTxBroadcastConfigError = errors.New("Failed building tx")
	// ErrFailedTxBroadcast is returned when transaction broadcast fails.
	ErrTxBroadcastNetworkError = errors.New("Failed broadcasting tx")
	// ErrTxBroadcastTimeout is returned when transaction broadcast times out.
	ErrTxBroadcastTimeout = errors.New("Broadcast timeout error")
	// ErrUnableToGetProof is returned when proof is not available.
	ErrUnableToGetProof = errors.New("Unable to get proof")
	// ErrRetrieval is returned when retrieval rpc falls
	ErrRetrieval = errors.New("Retrieval failed")
	// ErrBlobNotFound is returned when blob is not found.
	ErrBlobNotFound = errors.New("Blob not found")
	// ErrBlobNotIncluded is returned when blob is not included.
	ErrBlobNotIncluded = errors.New("Blob not included")
	// ErrProofNotMatching is returned when proof does not match.
	ErrProofNotMatching = errors.New("Proof not matching")
)
