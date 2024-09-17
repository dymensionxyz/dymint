package types

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/fraud"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNoStateFound          = errors.New("no state found")
	ErrEmptyBlock            = errors.New("block has no transactions and is not allowed to be empty")
	ErrInvalidBlockHeight    = errors.New("invalid block height")
	ErrInvalidHeaderDataHash = errors.New("header not matching block data")
	ErrMissingProposerPubKey = fmt.Errorf("missing proposer public key: %w", gerrc.ErrNotFound)
)

type ErrFraudHeightMismatch struct {
	expected uint64
	actual   uint64

	headerHeight uint64
	headerHash   [32]byte
	proposer     []byte
}

// NewErrFraudHeightMismatch creates a new ErrFraudHeightMismatch error.
func NewErrFraudHeightMismatch(expected uint64, actual uint64, block *Block) error {
	return &ErrFraudHeightMismatch{
		expected: expected, actual: actual,
		headerHeight: block.Header.Height, headerHash: block.Header.Hash(), proposer: block.Header.ProposerAddress,
	}
}

func (e ErrFraudHeightMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" height mismatch: state expected %d, got %d", e.headerHeight, e.headerHash, e.proposer, e.expected, e.actual)
}

func (e ErrFraudHeightMismatch) Unwrap() error {
	return fraud.ErrFraud
}

type ErrFraudAppHashMismatch struct {
	expected [32]byte

	headerHeight uint64
	headerHash   [32]byte
	proposer     []byte
}

// NewErrFraudAppHashMismatch creates a new ErrFraudAppHashMismatch error.
func NewErrFraudAppHashMismatch(expected [32]byte, actual [32]byte, block *Block) error {
	return &ErrFraudAppHashMismatch{
		expected:     expected,
		headerHeight: block.Header.Height, headerHash: block.Header.Hash(), proposer: block.Header.ProposerAddress,
	}
}

func (e ErrFraudAppHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" AppHash mismatch: state expected %X, got %X", e.headerHeight, e.headerHash, e.proposer, e.expected, e.headerHash)
}

func (e ErrFraudAppHashMismatch) Unwrap() error {
	return fraud.ErrFraud
}

type ErrLastResultsHashMismatch struct {
	expected [32]byte

	headerHeight   uint64
	headerHash     [32]byte
	proposer       []byte
	lastResultHash [32]byte
}

// NewErrLastResultsHashMismatch creates a new ErrLastResultsHashMismatch error.
func NewErrLastResultsHashMismatch(expected [32]byte, block *Block) error {
	return &ErrLastResultsHashMismatch{
		expected:     expected,
		headerHeight: block.Header.Height, headerHash: block.Header.Hash(), proposer: block.Header.ProposerAddress,
		lastResultHash: block.Header.LastResultsHash,
	}
}

func (e ErrLastResultsHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" LastResultsHash mismatch: state expected %X, got %X", e.headerHeight, e.headerHash, e.proposer, e.expected, e.lastResultHash)
}

func (e ErrLastResultsHashMismatch) Unwrap() error {
	return fraud.ErrFraud
}
