package types

import "errors"

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNoStateFound          = errors.New("no state found")
	ErrEmptyBlock            = errors.New("block has no transactions and is not allowed to be empty")
	ErrInvalidBlockHeight    = errors.New("invalid block height")
	ErrInvalidHeaderDataHash = errors.New("header not matching block data")
)
