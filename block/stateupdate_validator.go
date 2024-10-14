package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// StateUpdateValidator is a validator for messages gossiped in the p2p network.
type StateUpdateValidator struct {
	logger       types.Logger
	blockManager *Manager
}

// NewValidator creates a new Validator.
func NewStateUpdateValidator(logger types.Logger, blockManager *Manager) *StateUpdateValidator {
	return &StateUpdateValidator{
		logger:       logger,
		blockManager: blockManager,
	}
}

func (v *StateUpdateValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	v.logger.Debug("validating state update", "start height", batch.StartHeight, "end height", batch.EndHeight)

	var daBlocks []*types.Block
	var p2pBlocks []*types.Block

	// load blocks for the batch height, either P2P or DA blocks
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			continue
		}
		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			continue
		}
		if source == types.DA.String() {
			daBlocks = append(daBlocks, block)
		} else {
			p2pBlocks = append(p2pBlocks, block)
		}
	}

	if uint64(len(daBlocks)) != batch.NumBlocks {
		daBlocks = []*types.Block{}
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

		}
		for _, batch := range daBatch.Batches {
			daBlocks = append(daBlocks, batch.Blocks...)
		}
	}

	// validate DA blocks against the state update
	err := v.ValidateDaBlocks(batch, daBlocks)
	if err != nil {
		return err
	}

	// compare the batch blocks with the blocks applied from P2P
	err = v.ValidateP2PBlocks(daBlocks, p2pBlocks)
	if err != nil {
		return err
	}

	// update the last validated height to the batch last block height
	v.blockManager.State.SetLastValidatedHeight(batch.EndHeight)
	return nil
}

func (v *StateUpdateValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks []*types.Block) error {
	// nothing to compare
	if len(p2pBlocks) == 0 {
		return nil
	}

	// iterate over daBlocks and compare hashes if there block is also in p2pBlocks
	i := 0
	for _, daBlock := range daBlocks {

		if p2pBlocks[i].Header.Height != daBlock.Header.Height {
			break
		}
		p2pBlockHash, err := blockHash(p2pBlocks[i])
		if err != nil {
			return err
		}
		daBlockHash, err := blockHash(daBlock)
		if err != nil {
			return err
		}
		if !bytes.Equal(p2pBlockHash, daBlockHash) {
			return types.NewErrStateUpdateDoubleSigningFraud(daBlock.Header.Height)
		}
		i++
		if i == len(p2pBlocks) {
			break
		}
	}
	return nil
}

func (v *StateUpdateValidator) ValidateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error {
	// check numblocks
	numSlBDs := uint64((len(slBatch.BlockDescriptors)))
	numDABlocks := uint64(len(daBlocks))
	numSLBlocks := slBatch.NumBlocks
	if numSLBlocks != numDABlocks || numSLBlocks != numSlBDs {
		return types.NewErrStateUpdateNumBlocksNotMatchingFraud(slBatch.EndHeight, numSLBlocks, numDABlocks, numSLBlocks)
	}

	// check blocks
	for i, bd := range slBatch.BlockDescriptors {
		// height check
		if bd.Height != daBlocks[i].Header.Height {
			return types.NewErrStateUpdateHeightNotMatchingFraud(slBatch.StateIndex, bd.Height, daBlocks[i].Header.Height)
		}
		// we compare the state root between SL state info and DA block
		if !bytes.Equal(bd.StateRoot, daBlocks[i].Header.AppHash[:]) {
			return types.NewErrStateUpdateStateRootNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.StateRoot, daBlocks[i].Header.AppHash[:])
		}

		// we compare the timestamp between SL state info and DA block
		if !bd.Timestamp.Equal(daBlocks[i].Header.GetTimestamp()) {
			return types.NewErrStateUpdateTimestampNotMatchingFraud(slBatch.StateIndex, bd.Height, bd.Timestamp, daBlocks[i].Header.GetTimestamp())
		}

		err := v.validateDRS(slBatch.StateIndex, bd.Height, bd.DrsVersion)
		if err != nil {
			return err
		}

	}

	// TODO(srene): implement sequencer address validation
	return nil
}

// TODO(srene): implement DRS/height verification
func (v *StateUpdateValidator) validateDRS(stateIndex uint64, height uint64, version string) error {
	drs, err := v.blockManager.State.GetDRSVersion(height)
	if err != nil {
		panic(fmt.Errorf("unable to validate drs version. DRS: %s. height: %d err: %w", drs, height, err))
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
