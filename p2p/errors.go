package p2p

import "errors"

var (
	ErrNoPrivKey    = errors.New("private key not provided")
	ErrorInvalidCid = errors.New("invalid cid")
)
