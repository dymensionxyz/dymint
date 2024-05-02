package settlement

import (
	"fmt"

	"github.com/dymensionxyz/dymint/gerr"
)

var (
	// ErrBatchNotFound is returned when a batch is not found for the rollapp.
	ErrBatchNotFound = fmt.Errorf("batch not found: %w", gerr.ErrNotFound)
	// ErrEmptyResponse is returned when the response is empty.
	ErrEmptyResponse = fmt.Errorf("empty response: %w", gerr.ErrUnknown)
	// ErrNoSequencerForRollapp is returned when a sequencer is not found for the rollapp.
	ErrNoSequencerForRollapp = fmt.Errorf("no sequencer registered on the hub for this rollapp: %w", gerr.ErrNotFound)
	// ErrBatchNotAccepted is returned when a batch is not accepted by the settlement layer.
	ErrBatchNotAccepted = fmt.Errorf("batch not accepted: %w", gerr.ErrUnknown)
)
