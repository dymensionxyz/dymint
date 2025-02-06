package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	tmcrypto "github.com/tendermint/tendermint/crypto"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	ErrInvalidSignature      = errors.New("invalid signature")
	ErrNoStateFound          = errors.New("no state found")
	ErrEmptyBlock            = errors.New("block has no transactions and is not allowed to be empty")
	ErrInvalidBlockHeight    = errors.New("invalid block height")
	ErrInvalidHeaderDataHash = errors.New("header not matching block data")
	ErrInvalidDymHeader      = gerrc.ErrInvalidArgument.Wrap("dym header")
	ErrInvalidDymHeaderHash  = fmt.Errorf("hash mismatch: %w", ErrInvalidDymHeader)
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
func NewErrFraudHeightMismatch(expected uint64, header *Header) error {
	return &ErrFraudHeightMismatch{
		Expected:   expected,
		Actual:     header.Height,
		HeaderHash: header.Hash(),
		Proposer:   header.ProposerAddress,
	}
}

func (e ErrFraudHeightMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" height mismatch: state expected %d, got %d", e.Expected, e.HeaderHash, e.Proposer, e.Expected, e.Actual)
}

func (e ErrFraudHeightMismatch) Unwrap() error {
	return gerrc.ErrFault
}

// ErrFraudAppHashMismatch is the fraud that occurs when the AppHash of the block is different from the expected AppHash.
type ErrFraudAppHashMismatch struct {
	Expected [32]byte

	HeaderHeight uint64
	HeaderHash   [32]byte
	AppHash      [32]byte
	Proposer     []byte
}

// NewErrFraudAppHashMismatch creates a new ErrFraudAppHashMismatch error.
func NewErrFraudAppHashMismatch(expected [32]byte, header *Header) error {
	return &ErrFraudAppHashMismatch{
		Expected:     expected,
		HeaderHeight: header.Height,
		HeaderHash:   header.Hash(),
		AppHash:      header.AppHash,
		Proposer:     header.ProposerAddress,
	}
}

func (e ErrFraudAppHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" AppHash mismatch: state expected %X, got %X", e.HeaderHeight, e.HeaderHash, e.Proposer, e.Expected, e.AppHash)
}

func (e ErrFraudAppHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

// ErrLastResultsHashMismatch indicates a potential fraud when the LastResultsHash of a block does not match the expected value.
type ErrLastResultsHashMismatch struct {
	Expected [32]byte

	HeaderHeight   uint64
	HeaderHash     [32]byte
	Proposer       []byte
	LastResultHash [32]byte
}

// NewErrLastResultsHashMismatch creates a new ErrLastResultsHashMismatch error.
func NewErrLastResultsHashMismatch(expected [32]byte, header *Header) error {
	return &ErrLastResultsHashMismatch{
		Expected:       expected,
		HeaderHeight:   header.Height,
		HeaderHash:     header.Hash(),
		Proposer:       header.ProposerAddress,
		LastResultHash: header.LastResultsHash,
	}
}

func (e ErrLastResultsHashMismatch) Error() string {
	return fmt.Sprintf("possible fraud detected on height %d, with header hash %X, emitted by sequencer %X:"+
		" LastResultsHash mismatch: state expected %X, got %X", e.HeaderHeight, e.HeaderHash, e.Proposer, e.Expected, e.LastResultHash)
}

func (e ErrLastResultsHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

// ErrTimeFraud represents an error indicating a possible fraud due to time drift.
type ErrTimeFraud struct {
	Drift           time.Duration
	ProposerAddress []byte
	HeaderHash      [32]byte
	HeaderHeight    uint64
	HeaderTime      time.Time
	CurrentTime     time.Time
}

func NewErrTimeFraud(block *Block, currentTime time.Time) error {
	headerTime := block.Header.GetTimestamp()
	drift := headerTime.Sub(currentTime)

	return ErrTimeFraud{
		CurrentTime:     currentTime,
		Drift:           drift,
		ProposerAddress: block.Header.ProposerAddress,
		HeaderHash:      block.Header.Hash(),
		HeaderHeight:    block.Header.Height,
		HeaderTime:      headerTime,
	}
}

func (e ErrTimeFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: "+
			"Time drift exceeded: drift=%s, max allowed drift=%s, header time=%s, current time=%s",
		e.HeaderHeight, e.HeaderHash, e.ProposerAddress,
		e.Drift, TimeFraudMaxDrift, e.HeaderTime, e.CurrentTime,
	)
}

func (e ErrTimeFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrLastHeaderHashMismatch is the error that occurs when the last header hash does not match the expected value.
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
	return fmt.Sprintf("possible fraud detected: last header hash mismatch. expected=%X, got=%X", e.Expected, e.LastHeaderHash)
}

func (e ErrLastHeaderHashMismatch) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidChainID is the fraud that occurs when the chain ID of the block is different from the expected chain ID.
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
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: "+
			"Invalid Chain ID: expected %s, got %s",
		e.Block.Header.Height, e.Block.Header.Hash(), e.Block.Header.ProposerAddress,
		e.Expected, e.Block.Header.ChainID,
	)
}

func (e ErrInvalidChainID) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidBlockHeightFraud is the fraud that happens when the height that is on the commit header is
// different from the height of the block.
type ErrInvalidBlockHeightFraud struct {
	Expected uint64
	Header   *Header
}

func NewErrInvalidCommitBlockHeightFraud(expected uint64, header *Header) error {
	return &ErrInvalidBlockHeightFraud{
		Expected: expected,
		Header:   header,
	}
}

func (e ErrInvalidBlockHeightFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: "+
			"Invalid Block Height: expected %d, got %d",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.Expected, e.Header.Height,
	)
}

func (e ErrInvalidBlockHeightFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidHeaderHashFraud indicates a potential fraud when the Header Hash does not match the expected value.
type ErrInvalidHeaderHashFraud struct {
	ExpectedHash [32]byte
	Header       *Header
}

func NewErrInvalidHeaderHashFraud(expectedHash [32]byte, header *Header) error {
	return &ErrInvalidHeaderHashFraud{
		ExpectedHash: expectedHash,
		Header:       header,
	}
}

func (e ErrInvalidHeaderHashFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: "+
			"Invalid Header Hash: expected %X, got %X",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.ExpectedHash, e.Header.Hash(),
	)
}

func (e ErrInvalidHeaderHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidSignatureFraud indicates a potential fraud due to an invalid signature in the block.
type ErrInvalidSignatureFraud struct {
	Err    error
	Header *Header
	Commit *Commit
}

func NewErrInvalidSignatureFraud(err error, header *Header, c *Commit) error {
	return &ErrInvalidSignatureFraud{
		Header: header,
		Err:    err,
		Commit: c,
	}
}

func (e ErrInvalidSignatureFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: Invalid Signature: %s, signatures=%v",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.Err, e.Commit.Signatures,
	)
}

func (e ErrInvalidSignatureFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidProposerAddressFraud indicates a potential fraud when the proposer's address is invalid.
type ErrInvalidProposerAddressFraud struct {
	ExpectedAddress []byte
	ActualAddress   tmcrypto.Address

	Header *Header
}

func NewErrInvalidProposerAddressFraud(address []byte, address2 tmcrypto.Address, header *Header) error {
	return &ErrInvalidProposerAddressFraud{
		ExpectedAddress: address,
		ActualAddress:   address2,
		Header:          header,
	}
}

func (e ErrInvalidProposerAddressFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: Invalid Proposer Address: expected %X, got %X",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.ExpectedAddress, e.ActualAddress,
	)
}

func (e ErrInvalidProposerAddressFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidSequencerHashFraud indicates a potential fraud when the sequencer's hash is invalid.
type ErrInvalidSequencerHashFraud struct {
	ExpectedHash [32]byte
	ActualHash   []byte

	Header *Header
}

func NewErrInvalidSequencerHashFraud(expectedHash [32]byte, actualHash []byte, header *Header) error {
	return &ErrInvalidSequencerHashFraud{
		ExpectedHash: expectedHash,
		ActualHash:   actualHash,
		Header:       header,
	}
}

func (e ErrInvalidSequencerHashFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: Invalid Sequencer Hash: expected %X, got %X",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.ExpectedHash, e.ActualHash,
	)
}

func (e ErrInvalidSequencerHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidNextSequencersHashFraud indicates a potential fraud when the NextSequencersHash does not match the expected value.
type ErrInvalidNextSequencersHashFraud struct {
	ExpectedHash [32]byte
	Header       Header
}

func NewErrInvalidNextSequencersHashFraud(expectedHash [32]byte, header Header) error {
	return &ErrInvalidNextSequencersHashFraud{
		ExpectedHash: expectedHash,
		Header:       header,
	}
}

func (e ErrInvalidNextSequencersHashFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: Invalid Next Sequencers Hash: expected %X, got %X",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.ExpectedHash, e.Header.NextSequencersHash,
	)
}

func (e ErrInvalidNextSequencersHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrInvalidHeaderDataHashFraud indicates a potential fraud when the Header Data Hash does not match the expected value.
type ErrInvalidHeaderDataHashFraud struct {
	Expected [32]byte
	Actual   [32]byte

	Header Header
}

func NewErrInvalidHeaderDataHashFraud(block *Block) error {
	return &ErrInvalidHeaderDataHashFraud{
		Expected: block.Header.DataHash,
		Actual:   [32]byte(GetDataHash(block)),
		Header:   block.Header,
	}
}

func (e ErrInvalidHeaderDataHashFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d, with header hash %X, emitted by sequencer %X: Invalid Header Data Hash: expected %X, got %X",
		e.Header.Height, e.Header.Hash(), e.Header.ProposerAddress,
		e.Expected, e.Actual,
	)
}

func (e ErrInvalidHeaderDataHashFraud) Unwrap() error {
	return gerrc.ErrFault
}

// when dym header on the block header is not correctly derived from block data
type ErrInvalidDymHeaderFraud struct {
	*Block
	base error
}

func NewErrInvalidDymHeaderFraud(block *Block, err error) error {
	return &ErrInvalidDymHeaderFraud{
		Block: block,
		base:  err,
	}
}

func (e ErrInvalidDymHeaderFraud) Error() string {
	return fmt.Errorf(
		"possible fraud, double check using correct DRS and block version: invalid dym header: height : %d: %w",
		e.Header.Height,
		e.Unwrap(),
	).Error()
}

func (e ErrInvalidDymHeaderFraud) Unwrap() error {
	return errors.Join(e.base, gerrc.ErrFault)
}

// ErrStateUpdateNumBlocksNotMatchingFraud represents an error where the number of blocks in the state update does not match the expected number.
type ErrStateUpdateNumBlocksNotMatchingFraud struct {
	StateIndex  uint64
	SLNumBlocks uint64
	NumBds      uint64
	NumDABlocks uint64
}

func NewErrStateUpdateNumBlocksNotMatchingFraud(stateIndex, slNumBlocks, numbds, numDABlocks uint64) error {
	return &ErrStateUpdateNumBlocksNotMatchingFraud{
		StateIndex:  stateIndex,
		SLNumBlocks: slNumBlocks,
		NumBds:      numbds,
		NumDABlocks: numDABlocks,
	}
}

func (e ErrStateUpdateNumBlocksNotMatchingFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: Blocks Not Matching - StateIndex: %d, BatchNumBlocks: %d, NumBlockDescriptors: %d, NumDABlocks: %d",
		e.StateIndex, e.SLNumBlocks, e.NumBds, e.NumDABlocks,
	)
}

func (e ErrStateUpdateNumBlocksNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateHeightNotMatchingFraud is the fraud that happens when the height that is on the commit header is
// different from the height of the block.
type ErrStateUpdateHeightNotMatchingFraud struct {
	StateIndex    uint64
	SLBeginHeight uint64
	DABeginHeight uint64
	SLEndHeight   uint64
	DAEndHeight   uint64
}

func NewErrStateUpdateHeightNotMatchingFraud(stateIndex uint64, slBeginHeight uint64, daBeginHeight uint64, slEndHeight uint64, daEndHeight uint64) error {
	return &ErrStateUpdateHeightNotMatchingFraud{
		StateIndex:    stateIndex,
		SLBeginHeight: slBeginHeight,
		DABeginHeight: daBeginHeight,
		SLEndHeight:   slEndHeight,
		DAEndHeight:   daEndHeight,
	}
}

func (e ErrStateUpdateHeightNotMatchingFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: Height Mismatch - StateIndex: %d, SL Begin Height: %d, DA Begin Height: %d, SL End Height: %d, DA End Height: %d",
		e.StateIndex, e.SLBeginHeight, e.DABeginHeight, e.SLEndHeight, e.DAEndHeight,
	)
}

func (e ErrStateUpdateHeightNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateStateRootNotMatchingFraud represents an error where the state roots do not match in the state update.
type ErrStateUpdateStateRootNotMatchingFraud struct {
	StateIndex  uint64
	Height      uint64
	SLStateRoot []byte
	DAStateRoot []byte
}

func NewErrStateUpdateStateRootNotMatchingFraud(stateIndex uint64, height uint64, slStateRoot []byte, daStateRoot []byte) error {
	return &ErrStateUpdateStateRootNotMatchingFraud{
		StateIndex:  stateIndex,
		Height:      height,
		SLStateRoot: slStateRoot,
		DAStateRoot: daStateRoot,
	}
}

func (e ErrStateUpdateStateRootNotMatchingFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected on height %d: State Root Mismatch - StateIndex: %d, SL State Root: %X, DA State Root: %X",
		e.Height,
		e.StateIndex, e.SLStateRoot, e.DAStateRoot,
	)
}

func (e ErrStateUpdateStateRootNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateTimestampNotMatchingFraud represents an error where the timestamps do not match in the state update.
type ErrStateUpdateTimestampNotMatchingFraud struct {
	StateIndex  uint64
	Height      uint64
	SLTimestamp time.Time
	DATimestamp time.Time
}

func NewErrStateUpdateTimestampNotMatchingFraud(stateIndex uint64, height uint64, slTimestamp time.Time, daTimestamp time.Time) error {
	return &ErrStateUpdateTimestampNotMatchingFraud{
		StateIndex:  stateIndex,
		Height:      height,
		SLTimestamp: slTimestamp,
		DATimestamp: daTimestamp,
	}
}

func (e ErrStateUpdateTimestampNotMatchingFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: Timestamp Mismatch - StateIndex: %d, SL Timestamp: %s, DA Timestamp: %s",
		e.StateIndex, e.SLTimestamp.Format(time.RFC3339), e.DATimestamp.Format(time.RFC3339),
	)
}

func (e ErrStateUpdateTimestampNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateDoubleSigningFraud indicates a potential fraud due to double signing detected between DA and P2P blocks.
type ErrStateUpdateDoubleSigningFraud struct {
	DABlock      *Block
	P2PBlock     *Block
	DABlockHash  []byte
	P2PBlockHash []byte
}

func NewErrStateUpdateDoubleSigningFraud(daBlock *Block, p2pBlock *Block, daBlockHash []byte, p2pBlockHash []byte) error {
	return &ErrStateUpdateDoubleSigningFraud{
		DABlock:      daBlock,
		P2PBlock:     p2pBlock,
		DABlockHash:  daBlockHash,
		P2PBlockHash: p2pBlockHash,
	}
}

func (e ErrStateUpdateDoubleSigningFraud) Error() string {
	jsonDABlock, err := getJsonFromBlock(e.DABlock)
	if err != nil {
		return fmt.Sprintf("err json da block:%s", err)
	}
	jsonP2PBlock, err := getJsonFromBlock(e.P2PBlock)
	if err != nil {
		return fmt.Sprintf("err json p2p block:%s", err)
	}
	p2pBlockBytes, err := e.P2PBlock.MarshalBinary()
	if err != nil {
		return fmt.Sprintf("err marshal p2p block:%s", err)
	}
	daBlockBytes, err := e.DABlock.MarshalBinary()
	if err != nil {
		return fmt.Sprintf("err marshal da block:%s", err)
	}
	return fmt.Sprintf("possible fraud detected: block received from P2P not matching block found in DA. \n  P2P Json Block: %s \n DA Json Block:%s \n P2P Block hash:%s \n  DA block hash:%s \n  P2P block bytes:%s \n DA Block bytes:%s \n", jsonP2PBlock, jsonDABlock, hex.EncodeToString(e.P2PBlockHash), hex.EncodeToString(e.DABlockHash), hex.EncodeToString(p2pBlockBytes), hex.EncodeToString(daBlockBytes))
}

func (e ErrStateUpdateDoubleSigningFraud) Unwrap() error {
	return gerrc.ErrFault
}

func getJsonFromBlock(block *Block) ([]byte, error) {
	hash := block.Hash()
	abciBlock, err := ToABCIBlock(block)
	if err != nil {
		return nil, err
	}
	resultBlock := &ctypes.ResultBlock{
		BlockID: tmtypes.BlockID{
			Hash: hash[:],
			PartSetHeader: tmtypes.PartSetHeader{
				Total: 1,
				Hash:  hash[:],
			},
		},
		Block: abciBlock,
	}

	jsonBlock, err := legacy.Cdc.MarshalJSON(resultBlock)
	if err != nil {
		return nil, err
	}
	return jsonBlock, nil
}

// ErrStateUpdateBlobNotAvailableFraud represents an error where a blob is not available in DA.
type ErrStateUpdateBlobNotAvailableFraud struct {
	StateIndex uint64
	DA         string
	DAPath     string
}

func NewErrStateUpdateBlobNotAvailableFraud(stateIndex uint64, da string, daPath string) error {
	return &ErrStateUpdateBlobNotAvailableFraud{
		StateIndex: stateIndex,
		DA:         da,
		DAPath:     daPath,
	}
}

func (e ErrStateUpdateBlobNotAvailableFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: Blob Not Available in DA - StateIndex: %d, DA: %s, DAPath: %s",
		e.StateIndex, e.DA, e.DAPath,
	)
}

func (e ErrStateUpdateBlobNotAvailableFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateBlobCorruptedFraud represents an error where a blob is corrupted in DA.
type ErrStateUpdateBlobCorruptedFraud struct {
	StateIndex uint64
	DA         string
	DAPath     string
}

func NewErrStateUpdateBlobCorruptedFraud(stateIndex uint64, da string, daPath string) error {
	return &ErrStateUpdateBlobCorruptedFraud{
		StateIndex: stateIndex,
		DA:         da,
		DAPath:     daPath,
	}
}

func (e ErrStateUpdateBlobCorruptedFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: Blob Corrupted in DA - StateIndex: %d, DA: %s, DAPath: %s",
		e.StateIndex, e.DA, e.DAPath,
	)
}

func (e ErrStateUpdateBlobCorruptedFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateBlobNotFoundFraud represents an error where DA tx/commitment exists but it contains no blob
type ErrStateUpdateBlobNotFoundFraud struct {
	StateIndex uint64
	DA         string
	DAPath     string
}

func NewErrStateUpdateBlobNotFoundFraud(stateIndex uint64, da string, daPath string) error {
	return &ErrStateUpdateBlobNotFoundFraud{
		StateIndex: stateIndex,
		DA:         da,
		DAPath:     daPath,
	}
}

func (e ErrStateUpdateBlobNotFoundFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: no blob data found in DA - StateIndex: %d, DA: %s, DAPath: %s",
		e.StateIndex, e.DA, e.DAPath,
	)
}

func (e ErrStateUpdateBlobNotFoundFraud) Unwrap() error {
	return gerrc.ErrFault
}

// ErrStateUpdateDRSVersionFraud represents an error where the DRS versions do not match in the state update.
type ErrStateUpdateDRSVersionFraud struct {
	StateIndex   uint64
	Height       uint64
	BlockVersion uint32
	SLVersion    uint32
}

func NewErrStateUpdateDRSVersionFraud(stateIndex uint64, height uint64, blockVersion uint32, slVersion uint32) error {
	return &ErrStateUpdateDRSVersionFraud{
		StateIndex:   stateIndex,
		Height:       height,
		BlockVersion: blockVersion,
		SLVersion:    slVersion,
	}
}

func (e ErrStateUpdateDRSVersionFraud) Error() string {
	return fmt.Sprintf(
		"possible fraud detected: DRS Version Mismatch - StateIndex: %d, Block DRS: %d, SL DRS: %d",
		e.StateIndex, e.BlockVersion, e.SLVersion,
	)
}

func (e ErrStateUpdateDRSVersionFraud) Unwrap() error {
	return gerrc.ErrFault
}
