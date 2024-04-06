package types

import "errors"

var (
	// ErrInvalidSignature is returned when a signature is invalid.
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrNoStateFound      = errors.New("no state found")
	ErrSkippedEmptyBlock = errors.New("skipped empty block")

	// ErrInvalidBlockHeight is returned when a block has an invalid height.
	ErrInvalidBlockHeight = errors.New("invalid block height")
)
