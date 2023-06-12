package types

import "errors"

var (
	// ErrInvalidSignature is returned when a signature is invalid.
	ErrInvalidSignature = errors.New("invalid signature")
	ErrNoStateFound     = errors.New("no state found")
)
