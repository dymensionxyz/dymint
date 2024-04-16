package types

import "errors"

var (
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrNoStateFound       = errors.New("no state found")
	ErrSkippedEmptyBlock  = errors.New("skipped empty block")
	ErrInvalidBlockHeight = errors.New("invalid block height")
)
