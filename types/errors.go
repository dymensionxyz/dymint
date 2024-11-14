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
	return fmt.Sprintf("blocks not matching. StateIndex: %d Batch numblocks: %d Num of block descriptors: %d Num of DA blocks: %d", e.StateIndex, e.SLNumBlocks, e.NumBds, e.NumDABlocks)
}

func (e ErrStateUpdateNumBlocksNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

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
	return fmt.Sprintf("block height in DA batch not matching SL batch height. StateIndex: %d SL Begin height: %d DA Begin height: %d SL End height:%d DA End height: %d", e.StateIndex, e.SLBeginHeight, e.DABeginHeight, e.SLEndHeight, e.DAEndHeight)
}

func (e ErrStateUpdateHeightNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

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
	return fmt.Sprintf("state root in DA batch block not matching state root in SL. StateIndex: %d Height: %d SL state root: %s DA state root: %s", e.StateIndex, e.Height, hex.EncodeToString(e.SLStateRoot), hex.EncodeToString(e.DAStateRoot))
}

func (e ErrStateUpdateStateRootNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

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
	return fmt.Sprintf("timestamp in DA batch block not matching timestamp in SL. StateIndex: %d Height: %d SL timestsamp: %s DA timestamp: %s", e.StateIndex, e.Height, e.SLTimestamp, e.DATimestamp)
}

func (e ErrStateUpdateTimestampNotMatchingFraud) Unwrap() error {
	return gerrc.ErrFault
}

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
	return fmt.Sprintf("block received from P2P not matching block found in DA. \n  P2P Json Block: %s \n DA Json Block:%s \n P2P Block hash:%s \n  DA block hash:%s \n  P2P block bytes:%s \n DA Block bytes:%s \n", jsonP2PBlock, jsonDABlock, hex.EncodeToString(e.P2PBlockHash), hex.EncodeToString(e.DABlockHash), hex.EncodeToString(p2pBlockBytes), hex.EncodeToString(daBlockBytes))
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

type ErrStateUpdateBlobNotAvailableFraud struct {
	StateIndex uint64
	DA         string
	DAHeight   uint64
	Commitment string
}

func NewErrStateUpdateBlobNotAvailableFraud(stateIndex uint64, da string, daHeight uint64, commitment string) error {
	return &ErrStateUpdateBlobNotAvailableFraud{
		StateIndex: stateIndex,
		DA:         da,
		DAHeight:   daHeight,
		Commitment: commitment,
	}
}

func (e ErrStateUpdateBlobNotAvailableFraud) Error() string {
	return fmt.Sprintf("blob not available in DA. StateIndex: %d DA: %s DA Height: %d Commitment: %s", e.StateIndex, e.DA, e.DAHeight, e.Commitment)
}

func (e ErrStateUpdateBlobNotAvailableFraud) Unwrap() error {
	return gerrc.ErrFault
}

type ErrStateUpdateBlobCorruptedFraud struct {
	StateIndex uint64
	DA         string
	DAHeight   uint64
	Commitment string
}

func NewErrStateUpdateBlobCorruptedFraud(stateIndex uint64, da string, daHeight uint64, commitment string) error {
	return &ErrStateUpdateBlobCorruptedFraud{
		StateIndex: stateIndex,
		DA:         da,
		DAHeight:   daHeight,
		Commitment: commitment,
	}
}

func (e ErrStateUpdateBlobCorruptedFraud) Error() string {
	return fmt.Sprintf("blob not parsable in DA. StateIndex: %d DA: %s DA Height: %d Commitment: %s", e.StateIndex, e.DA, e.DAHeight, e.Commitment)
}

func (e ErrStateUpdateBlobCorruptedFraud) Unwrap() error {
	return gerrc.ErrFault
}

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
	return fmt.Sprintf("drs version not matching. StateIndex: %d Height: %d Block DRS: %d SL DRS: %d", e.StateIndex, e.Height, e.BlockVersion, e.SLVersion)
}

func (e ErrStateUpdateDRSVersionFraud) Unwrap() error {
	return gerrc.ErrFault
}
