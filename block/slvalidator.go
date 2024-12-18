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

type SettlementValidator struct {
	logger              types.Logger
	blockManager        *Manager
	lastValidatedHeight atomic.Uint64
}

func NewSettlementValidator(logger types.Logger, blockManager *Manager) *SettlementValidator {
	lastValidatedHeight, err := blockManager.Store.LoadValidationHeight()
	if err != nil {
	}

	validator := &SettlementValidator{
		logger:       logger,
		blockManager: blockManager, // backward reference
	}
	validator.lastValidatedHeight.Store(lastValidatedHeight)

	return validator
}

func (v *SettlementValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	p2pBlocks := make(map[uint64]*types.Block)
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			continue
		}

		if source != types.Gossiped && source != types.BlockSync {
			continue
		}

		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			continue
		}
		p2pBlocks[block.Header.Height] = block
	}

	var daBatch da.ResultRetrieveBatch
	for {
		daBatch = v.blockManager.Retriever.RetrieveBatches(batch.MetaData.DA)
		if daBatch.Code == da.StatusSuccess {
			break
		}

		if errors.Is(daBatch.BaseResult.Error, da.ErrBlobNotParsed) {
			return types.NewErrStateUpdateBlobCorruptedFraud(batch.StateIndex, string(batch.MetaData.DA.Client), batch.MetaData.DA.Height, hex.EncodeToString(batch.MetaData.DA.Commitment))
		}

		checkBatchResult := v.blockManager.Retriever.CheckBatchAvailability(batch.MetaData.DA)
		if errors.Is(checkBatchResult.Error, da.ErrBlobNotIncluded) {
			return types.NewErrStateUpdateBlobNotAvailableFraud(batch.StateIndex, string(batch.MetaData.DA.Client), batch.MetaData.DA.Height, hex.EncodeToString(batch.MetaData.DA.Commitment))
		}

		continue
	}

	daBlocks := []*types.Block{}
	for _, batch := range daBatch.Batches {
		daBlocks = append(daBlocks, batch.Blocks...)
		types.LastReceivedDAHeightGauge.Set(float64(batch.EndHeight()))
	}

	err := v.ValidateDaBlocks(batch, daBlocks)
	if err != nil {
		return err
	}

	if len(p2pBlocks) == 0 {
		return nil
	}

	err = v.ValidateP2PBlocks(daBlocks, p2pBlocks)
	if err != nil {
		return err
	}

	return nil
}

func (v *SettlementValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks map[uint64]*types.Block) error {
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
	return nil
}

func (v *SettlementValidator) ValidateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error {
	numSlBDs := uint64(len(slBatch.BlockDescriptors))
	numSLBlocks := slBatch.NumBlocks
	numDABlocks := uint64(len(daBlocks))
	if numSLBlocks != numSlBDs || numDABlocks < numSlBDs {
		return types.NewErrStateUpdateNumBlocksNotMatchingFraud(slBatch.EndHeight, numSLBlocks, numSLBlocks, numDABlocks)
	}

	for i, bd := range slBatch.BlockDescriptors {

		if bd.Height != daBlocks[i].Header.Height {
			return types.NewErrStateUpdateHeightNotMatchingFraud(slBatch.StateIndex, slBatch.BlockDescriptors[0].Height, daBlocks[0].Header.Height, slBatch.BlockDescriptors[len(slBatch.BlockDescriptors)-1].Height, daBlocks[len(daBlocks)-1].Header.Height)
		}

		if !bytes.Equal(bd.StateRoot, daBlocks[i].Header.AppHash[:]) {
			return types.NewErrStateUpdateStateRootNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.StateRoot, daBlocks[i].Header.AppHash[:])
		}

		if !bd.Timestamp.Equal(daBlocks[i].Header.GetTimestamp()) {
			return types.NewErrStateUpdateTimestampNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.Timestamp, daBlocks[i].Header.GetTimestamp())
		}

		err := v.validateDRS(slBatch.StateIndex, bd.Height, bd.DrsVersion)
		if err != nil {
			return err
		}
	}

	lastDABlock := daBlocks[numSlBDs-1]

	if v.blockManager.State.RevisionStartHeight-1 == lastDABlock.Header.Height {
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

	return nil
}

func (v *SettlementValidator) UpdateLastValidatedHeight(height uint64) {
	for {
		curr := v.lastValidatedHeight.Load()
		if v.lastValidatedHeight.CompareAndSwap(curr, max(curr, height)) {
			_, err := v.blockManager.Store.SaveValidationHeight(v.GetLastValidatedHeight(), nil)
			if err != nil {
			}
			break
		}
	}
}

func (v *SettlementValidator) GetLastValidatedHeight() uint64 {
	return v.lastValidatedHeight.Load()
}

func (v *SettlementValidator) NextValidationHeight() uint64 {
	return v.lastValidatedHeight.Load() + 1
}

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

func blockHash(block *types.Block) ([]byte, error) {
	blockBytes, err := block.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error hashing block. err: %w", err)
	}
	h := sha256.New()
	h.Write(blockBytes)
	return h.Sum(nil), nil
}
