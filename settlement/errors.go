package settlement

import (
	"fmt"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// ErrBatchNotAccepted is returned when a batch is not accepted by the settlement layer.
var ErrBatchNotAccepted = fmt.Errorf("batch not accepted: %w", gerrc.ErrUnknown)

// ErrProposerIsSentinel is returned when a rollapp has no sequencer assigned.
var ErrProposerIsSentinel = fmt.Errorf("proposer is sentinel")

type ErrNextSequencerAddressFraud struct {
	Expected string
	Actual   string
}

func NewErrNextSequencerAddressFraud(expected string, actual string) *ErrNextSequencerAddressFraud {
	return &ErrNextSequencerAddressFraud{Expected: expected, Actual: actual}
}

func (e ErrNextSequencerAddressFraud) Error() string {
	return fmt.Sprintf("next sequencer address fraud: expected %s, got %s", e.Expected, e.Actual)
}

func (e ErrNextSequencerAddressFraud) Wrap(err error) error {
	return gerrc.ErrFault
}
