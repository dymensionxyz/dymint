package settlement

import "errors"

var (
	// ErrBatchNotFound is returned when a batch is not found in the kvstore.
	ErrBatchNotFound = errors.New("batch does not exist")
)
