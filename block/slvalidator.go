package block

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/metrics"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// SettlementValidator validates batches from settlement layer with the corresponding blocks from DA and P2P.
type SettlementValidator struct {
	logger       types.Logger
	blockManager *Manager

	// immutable: the height the node was started from
	trustedHeight         uint64
	lastValidatedHeightMu sync.Mutex

	// state root / app hash contained in the header of the last validated block
	// we cache it here to because we need it to produce tee attestations, and at the time
	// of the attestation request the block might have already been pruned
	lastValidatedHeightBlockHeaderAppHash []byte
}

// NewSettlementValidator returns a new StateUpdateValidator instance.
func NewSettlementValidator(logger types.Logger, blockManager *Manager) *SettlementValidator {
	v := &SettlementValidator{
		logger:        logger,
		blockManager:  blockManager,
		trustedHeight: blockManager.State.LastBlockHeight.Load(),
	}
	if blockManager.Conf.TeeEnabled {
		v.UpdateLastValidatedHeight(v.GetTrustedHeight(), true)
	}

	return v
}

// ValidateStateUpdate validates that the blocks from the state info are available in DA,
// that the information included in the Hub state info matches the blocks retrieved from DA
// and those blocks are the same that are obtained via P2P.
func (v *SettlementValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	v.logger.Debug("validating state update", "start height", batch.StartHeight, "end height", batch.EndHeight)

	daClient, err := v.blockManager.Store.LoadDA(batch.EndHeight)
	if err != nil {
		return err
	}
	if daClient != string(batch.MetaData.Client) {
		return types.NewErrStateUpdateDAFraud(batch.StateIndex, batch.EndHeight, daClient, string(batch.MetaData.Client))
	}

	// loads blocks applied from P2P, if any.
	p2pBlocks := make(map[uint64]*types.Block)
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			err = fmt.Errorf("validate state update: load block source: %w", err)
			panic(err)
		}

		// if block is not P2P block, skip
		if source != types.Gossiped && source != types.BlockSync {
			continue
		}

		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			err = fmt.Errorf("validate state update: load block: %w", err)
			panic(err)
		}
		p2pBlocks[block.Header.Height] = block
	}

	// load all DA blocks from the batch to be validated
	var daBatch da.ResultRetrieveBatch
	for {
		daBatch = v.blockManager.fetchBatch(batch.MetaData)
		if daBatch.Code == da.StatusSuccess {
			break
		}

		// fraud detected in case no data blob is found for the da tx or commitment
		if errors.Is(daBatch.BaseResult.Error, da.ErrBlobNotFound) {
			return types.NewErrStateUpdateBlobNotFoundFraud(batch.StateIndex, string(batch.MetaData.Client), batch.MetaData.DAPath)
		}

		// fraud detected in case blob data is retrieved but unable to get blocks from it.
		if errors.Is(daBatch.BaseResult.Error, da.ErrBlobNotParsed) {
			return types.NewErrStateUpdateBlobCorruptedFraud(batch.StateIndex, string(batch.MetaData.Client), batch.MetaData.DAPath)
		}

		retriever := v.blockManager.GetRetriever(batch.MetaData.Client)
		if retriever == nil {
			return fmt.Errorf("missing DA in config. DA: %s", batch.MetaData.Client)
		}
		// fraud detected in case availability checks fail and therefore there certainty the blob, according to the state update DA path, is not available.
		checkBatchResult := retriever.CheckBatchAvailability(batch.MetaData.DAPath)
		if errors.Is(checkBatchResult.Error, da.ErrBlobNotIncluded) {
			return types.NewErrStateUpdateBlobNotAvailableFraud(batch.StateIndex, string(batch.MetaData.Client), batch.MetaData.DAPath)
		}

		// FIXME: how to handle non-happy case? not returning error?
		continue
	}

	daBlocks := []*types.Block{}
	for _, batch := range daBatch.Batches {
		daBlocks = append(daBlocks, batch.Blocks...)
	}

	// validate DA blocks against the state update
	err = v.ValidateDaBlocks(batch, daBlocks)
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
			return types.NewErrStateUpdateDoubleSigningFraud(daBlock, p2pBlock, daBlockHash, p2pBlockHash)
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

	// we get revision for the next state update
	revision, err := v.blockManager.getRevisionFromSL(lastDABlock.Header.Height + 1)
	if err != nil {
		return err
	}
	// if lastDaBlock is previous block to fork, dont validate nextsequencerhash of last block because it will not match
	if revision.StartHeight-1 == lastDABlock.Header.Height {
		v.logger.Debug("DA blocks, previous to fork, validated successfully", "start height", daBlocks[0].Header.Height, "end height", daBlocks[len(daBlocks)-1].Header.Height)
		return nil
	}

	expectedNextSeqHash := lastDABlock.Header.SequencerHash
	if slBatch.NextSequencer != slBatch.Sequencer {
		nextSequencer, found := v.blockManager.Sequencers.GetByAddress(slBatch.NextSequencer)
		if !found {
			return fmt.Errorf("next sequencer not found")
		}
		copy(expectedNextSeqHash[:], nextSequencer.MustHash())
	}

	if !bytes.Equal(expectedNextSeqHash[:], lastDABlock.Header.NextSequencersHash[:]) {
		return types.NewErrInvalidNextSequencersHashFraud(expectedNextSeqHash, lastDABlock.Header)
	}

	v.logger.Debug("DA blocks validated successfully", "start height", daBlocks[0].Header.Height, "end height", daBlocks[len(daBlocks)-1].Header.Height)
	return nil
}

// UpdateLastValidatedHeight sets the height saved in the Store if it is higher than the existing height
// returns OK if the value was updated successfully or did not need to be updated
// force = true means to allow updating to a lower height than currently stored
func (v *SettlementValidator) UpdateLastValidatedHeight(height uint64, force bool) {
	v.lastValidatedHeightMu.Lock()
	defer v.lastValidatedHeightMu.Unlock()
	if !force && height <= v.GetLastValidatedHeight() {
		return
	}
	_, err := v.blockManager.Store.SaveValidationHeight(height, nil)
	if err != nil {
		v.logger.Error("update validation height: %w", err)
	}
	block, err := v.blockManager.Store.LoadBlock(height)
	if err != nil {
		err = fmt.Errorf("update last validated height: load block: %w", err)
		panic(err)
	}
	v.lastValidatedHeightBlockHeaderAppHash = block.Header.AppHash[:]

	metrics.LastValidatedHeight.Set(float64(height))
}

func (v *SettlementValidator) GetLastValidatedHeightBlockHeaderAppHash() []byte {
	return v.lastValidatedHeightBlockHeaderAppHash
}

func (v *SettlementValidator) GetLastValidatedHeight() uint64 {
	x, _ := v.blockManager.Store.LoadValidationHeight()
	return x
}

// NextValidationHeight returns the next height that needs to be validated with settlement state updates.
func (v *SettlementValidator) NextValidationHeight() uint64 {
	return v.GetLastValidatedHeight() + 1
}

func (v *SettlementValidator) GetTrustedHeight() uint64 {
	return v.trustedHeight
}

// validateDRS compares the DRS version stored for the specific height, obtained from rollapp params.
func (v *SettlementValidator) validateDRS(stateIndex uint64, height uint64, version uint32) error {
	drs, err := v.blockManager.Store.LoadDRSVersion(height)
	// it can happen DRS is not found for blocks applied previous to migration, in case of migration from 2D rollapps
	if errors.Is(err, gerrc.ErrNotFound) {
		v.logger.Error("Unable to validate BD DRS version. Height:%d Err:%w", height, err)
		return nil
	}
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
