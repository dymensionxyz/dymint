package block

import "errors"

var (
	ErrNonRecoverable  = errors.New("non recoverable")
	ErrRecoverable     = errors.New("recoverable")
	ErrDAMismatch      = errors.New("data availability client mismatch")
	ErrVersionMismatch = errors.New("binary version mismatch")
)
