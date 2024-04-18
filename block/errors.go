package block

import "errors"

var (
	ErrNonRecoverable = errors.New("non recoverable")
	ErrRecoverable    = errors.New("recoverable")
)
