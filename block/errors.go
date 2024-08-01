package block

import "errors"

var (
	ErrNonRecoverable = errors.New("non recoverable")
	ErrRecoverable    = errors.New("recoverable")
	ErrDAUpgrade      = errors.New("da upgrade")
	ErrVersionUpgrade = errors.New("version upgrade")
)
