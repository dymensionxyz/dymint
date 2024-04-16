package block

import "errors"

var (
	ErrProduceAndGossipBlockNonRecoverable = errors.New("non recoverable")
	ErrProduceAndGossipBlockRecoverable    = errors.New("recoverable")
)
