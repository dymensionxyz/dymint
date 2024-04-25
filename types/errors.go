package types

import "errors"

var (
	// ErrInvalidSignature is returned when a signature is invalid.
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrNoStateFound      = errors.New("no state found")
	ErrSkippedEmptyBlock = errors.New("skipped empty block")

	// fraud proofs..

	ErrInvalidISR            = errors.New("invalid ISR")
	ErrBlockISRCountMismatch = errors.New("block ISR count mismatch")
	ErrBlockMissingISR       = errors.New("block missing ISRs")
)
