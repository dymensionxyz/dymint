package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// SettlementValidator validates batches from settlement layer with the corresponding blocks from DA and P2P.
type SettlementValidator struct {
	logger              types.Logger
	blockManager        *Manager
	lastValidatedHeight atomic.Uint64
}

// NewSettlementValidator returns a new StateUpdateValidator instance.
func NewSettlementValidator(logger types.Logger, blockManager *Manager) *SettlementValidator {
	lastValidatedHeight, err := blockManager.Store.LoadValidationHeight()
	if err != nil {
		logger.Debug("validation height not loaded", "err", err)
	}

	validator := &SettlementValidator{
		logger:       logger,
		blockManager: blockManager,
	}
	validator.lastValidatedHeight.Store(lastValidatedHeight)

	return validator
}

// ValidateStateUpdate validates that the blocks from the state info are available in DA,
// that the information included in the Hub state info matches the blocks retrieved from DA
// and those blocks are the same that are obtained via P2P.
func (v *SettlementValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	v.logger.Debug("validating state update", "start height", batch.StartHeight, "end height", batch.EndHeight)

	// loads blocks applied from P2P, if any.
	p2pBlocks := make(map[uint64]*types.Block)
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			v.logger.Error("load block source", "error", err)
			continue
		}

		// if block is not P2P block, skip
		if source != types.Gossiped && source != types.BlockSync {
			continue
		}

		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			v.logger.Error("load block", "error", err)
			continue
		}
		p2pBlocks[block.Header.Height] = block
	}

	// load all DA blocks from the batch to be validated
	var daBatch da.ResultRetrieveBatch
	for {
		daBatch = v.blockManager.Retriever.RetrieveBatches(batch.MetaData.DA)
		if daBatch.Code == da.StatusSuccess {
			break
		}

		// fraud detected in case blob is retrieved but unable to get blocks from it.
		if errors.Is(daBatch.BaseResult.Error, da.ErrBlobNotParsed) {
			return types.NewErrStateUpdateBlobCorruptedFraud(batch.StateIndex, string(batch.MetaData.DA.Client), batch.MetaData.DA.Height, hex.EncodeToString(batch.MetaData.DA.Commitment))
		}

		// fraud detected in case availability checks fail and therefore there certainty the blob, according to the state update DA path, is not available.
		checkBatchResult := v.blockManager.Retriever.CheckBatchAvailability(batch.MetaData.DA)
		if errors.Is(checkBatchResult.Error, da.ErrBlobNotIncluded) {
			return types.NewErrStateUpdateBlobNotAvailableFraud(batch.StateIndex, string(batch.MetaData.DA.Client), batch.MetaData.DA.Height, hex.EncodeToString(batch.MetaData.DA.Commitment))
		}

		// FIXME: how to handle non-happy case? not returning error?
		continue
	}

	daBlocks := []*types.Block{}
	for _, batch := range daBatch.Batches {
		daBlocks = append(daBlocks, batch.Blocks...)
	}

	// validate DA blocks against the state update
	err := v.ValidateDaBlocks(batch, daBlocks)
	if err != nil {
		return err
	}

	// nothing to validate at P2P level, finish here.
	if len(p2pBlocks) == 0 {
		return nil
	}

	// validate P2P blocks against DA blocks
	err = v.ValidateP2PBlocks(daBlocks, p2pBlocks)
	if err != nil {
		return err
	}

	return nil
}

// ValidateP2PBlocks basically compares that the blocks applied from P2P are the same blocks included in the batch and retrieved from DA.
// Since DA blocks have been already validated against Hub state info block descriptors, if P2P blocks match with DA blocks, it means they are also validated against state info block descriptors.
func (v *SettlementValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks map[uint64]*types.Block) error {
	// iterate over daBlocks and compare hashes with the corresponding block from P2P (if exists) to see whether they are actually the same block
	for _, daBlock := range daBlocks {

		p2pBlock, ok := p2pBlocks[daBlock.Header.Height]
		if !ok {
			continue
		}
		p2pBlockHash, err := blockHash(p2pBlock)
		if err != nil {
			return err
		}
		daBlockHash, err := blockHash(daBlock)
		if err != nil {
			return err
		}
		if !bytes.Equal(p2pBlockHash, daBlockHash) {
			return types.NewErrStateUpdateDoubleSigningFraud(daBlock, p2pBlock)
		}

	}
	v.logger.Debug("P2P blocks validated successfully", "start height", daBlocks[0].Header.Height, "end height", daBlocks[len(daBlocks)-1].Header.Height)
	return nil
}

// ValidateDaBlocks checks that the information included in the Hub state info (height, state roots and timestamps), correspond to the blocks obtained from DA.
func (v *SettlementValidator) ValidateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error {
	// we first verify the numblocks included in the state info match the block descriptors and the blocks obtained from DA
	numSlBDs := uint64(len(slBatch.BlockDescriptors))
	numSLBlocks := slBatch.NumBlocks
	numDABlocks := uint64(len(daBlocks))
	if numSLBlocks != numSlBDs || numDABlocks < numSlBDs {
		return types.NewErrStateUpdateNumBlocksNotMatchingFraud(slBatch.EndHeight, numSLBlocks, numSLBlocks, numDABlocks)
	}

	// we compare all DA blocks against the information included in the state info block descriptors
	for i, bd := range slBatch.BlockDescriptors {
		// height check
		if bd.Height != daBlocks[i].Header.Height {
			return types.NewErrStateUpdateHeightNotMatchingFraud(slBatch.StateIndex, slBatch.BlockDescriptors[0].Height, daBlocks[0].Header.Height, slBatch.BlockDescriptors[len(slBatch.BlockDescriptors)-1].Height, daBlocks[len(daBlocks)-1].Header.Height)
		}
		// we compare the state root between SL state info and DA block
		if !bytes.Equal(bd.StateRoot, daBlocks[i].Header.AppHash[:]) {
			return types.NewErrStateUpdateStateRootNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.StateRoot, daBlocks[i].Header.AppHash[:])
		}

		// we compare the timestamp between SL state info and DA block
		if !bd.Timestamp.Equal(daBlocks[i].Header.GetTimestamp()) {
			return types.NewErrStateUpdateTimestampNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.Timestamp, daBlocks[i].Header.GetTimestamp())
		}

		// we validate block descriptor drs version per height
		err := v.validateDRS(slBatch.StateIndex, bd.Height, bd.DrsVersion)
		if err != nil {
			return err
		}
	}

	// we compare the sequencer address between SL state info and DA block
	// if next sequencer is not set, we check if the sequencer hash is equal to the next sequencer hash
	// because it did not change. If the next sequencer is set, we check if the next sequencer hash is equal on the
	// last block of the batch
	lastDABlock := daBlocks[numSlBDs-1]
	expectedNextSeqHash := lastDABlock.Header.SequencerHash
	if slBatch.NextSequencer != slBatch.Sequencer {
		nextSequencer, found := v.blockManager.Sequencers.GetByAddress(slBatch.NextSequencer)
		if !found {
			return fmt.Errorf("next sequencer not found")
		}
		copy(expectedNextSeqHash[:], nextSequencer.MustHash())
	}
	// FIXME: support hard fork
	if !bytes.Equal(expectedNextSeqHash[:], lastDABlock.Header.NextSequencersHash[:]) {
		return types.NewErrInvalidNextSequencersHashFraud(
			expectedNextSeqHash,
			lastDABlock.Header.NextSequencersHash,
		)
	}

	v.logger.Debug("DA blocks validated successfully", "start height", daBlocks[0].Header.Height, "end height", daBlocks[len(daBlocks)-1].Header.Height)
	return nil
}

// UpdateLastValidatedHeight sets the height saved in the Store if it is higher than the existing height
// returns OK if the value was updated successfully or did not need to be updated
func (v *SettlementValidator) UpdateLastValidatedHeight(height uint64) {
	for {
		curr := v.lastValidatedHeight.Load()
		if v.lastValidatedHeight.CompareAndSwap(curr, max(curr, height)) {
			_, err := v.blockManager.Store.SaveValidationHeight(v.GetLastValidatedHeight(), nil)
			if err != nil {
				v.logger.Error("update validation height: %w", err)
			}
			break
		}
	}
}

// GetLastValidatedHeight returns the most last block height that is validated with settlement state updates.
func (v *SettlementValidator) GetLastValidatedHeight() uint64 {
	return v.lastValidatedHeight.Load()
}

// NextValidationHeight returns the next height that needs to be validated with settlement state updates.
func (v *SettlementValidator) NextValidationHeight() uint64 {
	return v.lastValidatedHeight.Load() + 1
}

// validateDRS compares the DRS version stored for the specific height, obtained from rollapp params.
func (v *SettlementValidator) validateDRS(stateIndex uint64, height uint64, version uint32) error {
	drs, err := v.blockManager.Store.LoadDRSVersion(height)
	if err != nil {
		return err
	}
	if drs != version {
		return types.NewErrStateUpdateDRSVersionFraud(stateIndex, height, drs, version)
	}

	return nil
}

// blockHash generates a hash from the block bytes to compare them
func blockHash(block *types.Block) ([]byte, error) {
	blockBytes, err := block.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error hashing block. err: %w", err)
	}
	h := sha256.New()
	h.Write(blockBytes)
	return h.Sum(nil), nil
}
