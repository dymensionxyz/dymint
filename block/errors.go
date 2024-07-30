package block

import "errors"

var (
	ErrNonRecoverable = errors.New("non recoverable")
	ErrRecoverable    = errors.New("recoverable")
	ErrWrongDA        = errors.New("DA in config not matching DA path")
)
