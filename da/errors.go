package da

import "errors"

var (
	// ErrFailedTxBuild is returned when transaction build fails.
	ErrTxBroadcastConfigError = errors.New("Failed building tx")
	// ErrFailedTxBroadcast is returned when transaction broadcast fails.
	ErrTxBroadcastNetworkError = errors.New("Failed broadcasting tx")
	// ErrTxBroadcastTimeout is returned when transaction broadcast times out.
	ErrTxBroadcastTimeout = errors.New("Broadcast timeout error")
)
