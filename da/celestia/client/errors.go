package client

// Code defines error codes for JSON-RPC.
type Code int

// Codes are used by JSON-RPC client and server
const (
	CodeBlobNotFound               Code = 32001
	CodeBlobSizeOverLimit          Code = 32002
	CodeTxTimedOut                 Code = 32003
	CodeTxAlreadyInMempool         Code = 32004
	CodeTxIncorrectAccountSequence Code = 32005
	CodeTxTooLarge                 Code = 32006
	CodeContextDeadline            Code = 32007
	CodeFutureHeight               Code = 32008
	CodeHeightZero                 Code = 32009
)

// ErrBlobNotFound is used to indicate that the blob was not found.
type ErrBlobNotFound struct{}

func (e *ErrBlobNotFound) Error() string {
	return "blob: not found"
}

// ErrBlobSizeOverLimit is used to indicate that the blob size is over limit.
type ErrBlobSizeOverLimit struct{}

func (e *ErrBlobSizeOverLimit) Error() string {
	return "blob: over size limit"
}

// ErrTxTimedOut is the error message returned by the DA when mempool is congested.
type ErrTxTimedOut struct{}

func (e *ErrTxTimedOut) Error() string {
	return "timed out waiting for tx to be included in a block"
}

// ErrTxAlreadyInMempool is the error message returned by the DA when tx is already in mempool.
type ErrTxAlreadyInMempool struct{}

func (e *ErrTxAlreadyInMempool) Error() string {
	return "tx already in mempool"
}

// ErrTxIncorrectAccountSequence is the error message returned by the DA when tx has incorrect sequence.
type ErrTxIncorrectAccountSequence struct{}

func (e *ErrTxIncorrectAccountSequence) Error() string {
	return "incorrect account sequence"
}

// ErrTxTooLarge is the err message returned by the DA when tx size is too large.
type ErrTxTooLarge struct{}

func (e *ErrTxTooLarge) Error() string {
	return "tx too large"
}

// ErrContextDeadline is the error message returned by the DA when context deadline exceeds.
type ErrContextDeadline struct{}

func (e *ErrContextDeadline) Error() string {
	return "context deadline"
}

// ErrFutureHeight is returned when requested height is from the future
type ErrFutureHeight struct{}

func (e *ErrFutureHeight) Error() string {
	return "given height is from the future"
}

// ErrHeightZero returned when requested headers height is equal to 0.
type ErrHeightZero struct{}

func (e *ErrFutureHeight) Error() string {
	return "height is equal to 0"
}
