package fraud

import (
	"context"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
)

type Fault struct {
	Underlying error

	// Block defines the block in which the fault happened.
	Block *types.Block
}

func (f Fault) Error() string {
	return fmt.Sprintf("fraud error: %s", f.Underlying)
}

func (f Fault) Is(err error) bool {
	fault, ok := (err).(Fault)
	if !ok {
		return false
	}
	return errors.Is(fault.Underlying, f.Underlying)
}

func (f Fault) Unwrap() error {
	return f.Underlying
}

func (f Fault) GovProposalContent() string {
	return f.Error()
}

func NewFault(err error, block *types.Block) Fault {
	return Fault{Underlying: err, Block: block}
}

func IsFault(err error) bool {
	return errors.As(err, &Fault{})
}

type Handler interface {
	HandleFault(ctx context.Context, fault Fault)
}
