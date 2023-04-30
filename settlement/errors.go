package settlement

import "errors"

var (
	// ErrBatchNotFound is returned when a batch is not found for the rollapp.
	ErrBatchNotFound = errors.New("batch not found")
	// ErrNoSequencerForRollapp is returned when a sequencer is not found for the rollapp.
	ErrNoSequencerForRollapp = errors.New("no sequencer registered on the hub for this rollapp")
)
