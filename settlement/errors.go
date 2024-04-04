package settlement

import "errors"

var (
	// ErrBatchNotFound is returned when a batch is not found for the rollapp.
	ErrBatchNotFound = errors.New("batch not found")
	// ErrEmptyResponse is returned when the response is empty.
	ErrEmptyResponse = errors.New("empty response")
	// ErrNoSequencerForRollapp is returned when a sequencer is not found for the rollapp.
	ErrNoSequencerForRollapp = errors.New("no sequencer registered on the hub for this rollapp")
	// ErrBatchNotAccepted is returned when a batch is not accepted by the settlement layer.
	ErrBatchNotAccepted = errors.New("batch not accepted")
)
