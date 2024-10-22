package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	tmcrypto "github.com/tendermint/tendermint/crypto"
)

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNoStateFound          = errors.New("no state found")
	ErrEmptyBlock            = errors.New("block has no transactions and is not allowed to be empty")
	ErrInvalidBlockHeight    = errors.New("invalid block height")
	ErrInvalidHeaderDataHash = errors.New("header not matching block data")
	ErrMissingProposerPubKey = fmt.Errorf("missing proposer public key: %w", gerrc.ErrNotFound)
	ErrVersionMismatch       = errors.New("version mismatch")
	ErrEmptyProposerAddress  = errors.New("no proposer address")
)

// TimeFraudMaxDrift is the maximum allowed time drift between the block time and the local time.
var TimeFraudMaxDrift = 10 * time.Minute

// ErrFraudHeightMismatch is the fraud that occurs when the height of the block is different from the expected
// next height of the state.
type ErrFraudHeightMismatch struct {
	Expected uint64
	Actual   uint64

	HeaderHash [32]byte
	Proposer   []byte
}

// NewErrFraudHeightMismatch creates a new ErrFraudHeightMismatch error.
func NewErrFraudHeightMismatch(expected uint64, actual uint64, block *Block) error {
	return &ErrFraudHeightMismatch{
		Expected: expected, Actual: actual,
		HeaderHash: block.Header.Hash(), Proposer: block.Header.ProposerAddress,
	}
}

func (e ErrFraudHeightMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" height mismatch: state expected %d, got %d", e.Expected, e.HeaderHash, e.Proposer, e.Expected, e.Actual)
}

func (e ErrFraudHeightMismatch) Unwrap() error {
	return gerrc.ErrFault
}

type ErrFraudAppHashMismatch struct {
	Expected [32]byte

	HeaderHeight uint64
	HeaderHash   [32]byte
	Proposer     []byte
}

// NewErrFraudAppHashMismatch creates a new ErrFraudAppHashMismatch error.
func NewErrFraudAppHashMismatch(expected [32]byte, actual [32]byte, block *Block) error {
	return &ErrFraudAppHashMismatch{
		Expected:     expected,
		HeaderHeight: block.Header.Height, HeaderHash: block.Header.Hash(), Proposer: block.Header.ProposerAddress,
	}
}

func (e ErrFraudAppHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" AppHash mismatch: state expected %X, got %X", e.HeaderHeight, e.HeaderHash, e.Proposer, e.Expected, e.HeaderHash)
}

func (e ErrFraudAppHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

type ErrLastResultsHashMismatch struct {
	Expected [32]byte

	HeaderHeight   uint64
	HeaderHash     [32]byte
	Proposer       []byte
	LastResultHash [32]byte
}

// NewErrLastResultsHashMismatch creates a new ErrLastResultsHashMismatch error.
func NewErrLastResultsHashMismatch(expected [32]byte, block *Block) error {
	return &ErrLastResultsHashMismatch{
		Expected:     expected,
		HeaderHeight: block.Header.Height, HeaderHash: block.Header.Hash(), Proposer: block.Header.ProposerAddress,
		LastResultHash: block.Header.LastResultsHash,
	}
}

func (e ErrLastResultsHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" LastResultsHash mismatch: state expected %X, got %X", e.HeaderHeight, e.HeaderHash, e.Proposer, e.Expected, e.LastResultHash)
}

func (e ErrLastResultsHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

type ErrTimeFraud struct {
	Drift           time.Duration
	ProposerAddress []byte
	HeaderHash      [32]byte
	HeaderHeight    uint64
	HeaderTime      time.Time
	CurrentTime     time.Time
}

func NewErrTimeFraud(block *Block, currentTime time.Time) error {
	drift := time.Unix(int64(block.Header.Time), 0).Sub(currentTime)

	return ErrTimeFraud{
		Drift:           drift,
		ProposerAddress: block.Header.ProposerAddress,
		HeaderHash:      block.Header.Hash(),
		HeaderHeight:    block.Header.Height,
		HeaderTime:      time.Unix(int64(block.Header.Time), 0),
		CurrentTime:     currentTime,
	}
}

func (e ErrTimeFraud) Error() string {
	return fmt.Sprintf(
		"sequencer posted a block with invalid time. "+
			"Max allowed drift exceeded. "+
			"proposerAddress=%s headerHash=%s headerHeight=%d drift=%s MaxDrift=%s headerTime=%s currentTime=%s",
		e.ProposerAddress, e.HeaderHash, e.HeaderHeight, e.Drift, TimeFraudMaxDrift, e.HeaderTime, e.CurrentTime,
	)
}

func (e ErrTimeFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrLastHeaderHashMismatch struct {
	Expected       [32]byte
	LastHeaderHash [32]byte
}

func NewErrLastHeaderHashMismatch(expected [32]byte, block *Block) error {
	return &ErrLastHeaderHashMismatch{
		Expected:       expected,
		LastHeaderHash: block.Header.LastHeaderHash,
	}
}

func (e ErrLastHeaderHashMismatch) Error() string {
	return fmt.Sprintf("last header hash mismatch. expected=%X, got=%X", e.Expected, e.LastHeaderHash)
}

func (e ErrLastHeaderHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidChainID struct {
	Expected string
	Block    *Block
}

func NewErrInvalidChainID(expected string, block *Block) error {
	return &ErrInvalidChainID{
		Expected: expected,
		Block:    block,
	}
}

func (e ErrInvalidChainID) Error() string {
	return fmt.Sprintf("invalid chain ID. expected=%s, got=%s", e.Expected, e.Block.Header.ChainID)
}

func (e ErrInvalidChainID) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidBlockHeightFraud is the fraud that happens when the height that is on the commit header is
// different from the height of the block.
type ErrInvalidBlockHeightFraud struct {
	Expected     uint64
	ActualHeight uint64
}

func NewErrInvalidBlockHeightFraud(expected uint64, actualHeight uint64) error {
	return &ErrInvalidBlockHeightFraud{
		Expected:     expected,
		ActualHeight: actualHeight,
	}
}

func (e ErrInvalidBlockHeightFraud) Error() string {
	return fmt.Sprintf("invalid block height. expected=%d, got=%d", e.Expected, e.ActualHeight)
}

func (e ErrInvalidBlockHeightFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidHeaderHashFraud struct {
	ExpectedHash [32]byte
	ActualHash   [32]byte
}

func NewErrInvalidHeaderHashFraud(expectedHash [32]byte, actualHash [32]byte) error {
	return &ErrInvalidHeaderHashFraud{
		ExpectedHash: expectedHash,
		ActualHash:   actualHash,
	}
}

func (e ErrInvalidHeaderHashFraud) Error() string {
	return fmt.Sprintf("invalid header hash. expected=%X, got=%X", e.ExpectedHash, e.ActualHash)
}

func (e ErrInvalidHeaderHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidSignatureFraud struct {
	Err error
}

func NewErrInvalidSignatureFraud(err error) error {
	return &ErrInvalidSignatureFraud{
		Err: err,
	}
}

func (e ErrInvalidSignatureFraud) Error() string {
	return fmt.Sprintf("invalid signature: %s", e.Err)
}

func (e ErrInvalidSignatureFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidProposerAddressFraud struct {
	ExpectedAddress []byte
	ActualAddress   tmcrypto.Address
}

func NewErrInvalidProposerAddressFraud(address []byte, address2 tmcrypto.Address) error {
	return &ErrInvalidProposerAddressFraud{
		ExpectedAddress: address,
		ActualAddress:   address2,
	}
}

func (e ErrInvalidProposerAddressFraud) Error() string {
	return fmt.Sprintf("invalid proposer address. expected=%X, got=%X", e.ExpectedAddress, e.ActualAddress)
}

func (e ErrInvalidProposerAddressFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidSequencerHashFraud struct {
	ExpectedHash [32]byte
	ActualHash   []byte
}

func NewErrInvalidSequencerHashFraud(expectedHash [32]byte, actualHash []byte) error {
	return &ErrInvalidSequencerHashFraud{
		ExpectedHash: expectedHash,
		ActualHash:   actualHash,
	}
}

func (e ErrInvalidSequencerHashFraud) Error() string {
	return fmt.Sprintf("invalid sequencer hash. expected=%X, got=%X", e.ExpectedHash, e.ActualHash)
}

func (e ErrInvalidSequencerHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidNextSequencersHashFraud struct {
	ExpectedHash [32]byte
	ActualHash   [32]byte
}

func NewErrInvalidNextSequencersHashFraud(expectedHash [32]byte, actualHash [32]byte) error {
	return &ErrInvalidNextSequencersHashFraud{
		ExpectedHash: expectedHash,
		ActualHash:   actualHash,
	}
}

func (e ErrInvalidNextSequencersHashFraud) Error() string {
	return fmt.Sprintf("invalid next sequencers hash. expected=%X, got=%X", e.ExpectedHash, e.ActualHash)
}

func (e ErrInvalidNextSequencersHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrInvalidHeaderDataHashFraud struct {
	Expected [32]byte
	Actual   [32]byte
}

func NewErrInvalidHeaderDataHashFraud(expected [32]byte, actual [32]byte) error {
	return &ErrInvalidHeaderDataHashFraud{
		Expected: expected,
		Actual:   actual,
	}
}

func (e ErrInvalidHeaderDataHashFraud) Error() string {
	return fmt.Sprintf("invalid header data hash. expected=%X, got=%X", e.Expected, e.Actual)
}

func (e ErrInvalidHeaderDataHashFraud) Unwrap() error {
	return gerrc.ErrFault
}
