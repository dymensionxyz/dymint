package block

import (
	"bytes"
	"crypto/sha256"
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

// ValidateStateUpdate validates that the blocks from the state info are available in DA,
// that the information included in the Hub state info matches the blocks retrieved from DA
// and those blocks are the same that are obtained via P2P.
func (v *StateUpdateValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	v.logger.Debug("validating state update", "start height", batch.StartHeight, "end height", batch.EndHeight)

	// validate that the DRS version specified batch matches the node state
	err := v.validateDRS(batch.StartHeight, batch.EndHeight)
	if err != nil {
		return err
	}

	// load blocks applied from P2P for the batch heights
	p2pBlocks := make(map[uint64]*types.Block)
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			continue
		}
		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			continue
		}
		if source == types.Gossiped.String() || source == types.BlockSync.String() {
			p2pBlocks[block.Header.Height] = block
		}
	}

	// load all DA blocks from the batch to be validated
	daBlocks := []*types.Block{}
	var daBatch da.ResultRetrieveBatch
	for {
		daBatch = v.blockManager.Retriever.RetrieveBatches(batch.MetaData.DA)
		if daBatch.Code == da.StatusSuccess {
			break
		}
	}
	for _, batch := range daBatch.Batches {
		daBlocks = append(daBlocks, batch.Blocks...)
	}

	// validate DA blocks against the state update
	err = v.ValidateDaBlocks(batch, daBlocks)
	if err != nil {
		return err
	}

	// validate P2P blocks against DA blocks
	err = v.ValidateP2PBlocks(daBlocks, p2pBlocks)
	if err != nil {
		return err
	}

	// update the last validated height to the batch last block height
	v.blockManager.UpdateLastValidatedHeight(batch.EndHeight)

	return nil
}

// ValidateP2PBlocks basically compares that the blocks applied from P2P are the same blocks included in the batch and retrieved from DA.
// Since DA blocks have been already validated against Hub state info block descriptors, if P2P blocks match with DA blocks, it means they are also validated against state info block descriptors.
func (v *StateUpdateValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks map[uint64]*types.Block) error {
	// nothing to compare
	if len(p2pBlocks) == 0 {
		return nil
	}

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
			return types.NewErrStateUpdateDoubleSigningFraud(daBlock.Header.Height, p2pBlock)
		}

	}
	return nil
}

// ValidateDaBlocks checks that the information included in the Hub state info (height, state roots and timestamps), correspond to the blocks obtained from DA.
func (v *StateUpdateValidator) ValidateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error {
	// we first verify the numblocks included in the state info match the block descriptors and the blocks obtained from DA
	numSlBDs := uint64(len(slBatch.BlockDescriptors))
	numDABlocks := uint64(len(daBlocks))
	numSLBlocks := slBatch.NumBlocks
	if numSLBlocks != numDABlocks || numSLBlocks != numSlBDs {
		return types.NewErrStateUpdateNumBlocksNotMatchingFraud(slBatch.EndHeight, numSLBlocks, numDABlocks, numSLBlocks)
	}

	// we compare all DA blocks against the information included in the state info block descriptors
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
	}

	// TODO(srene): implement sequencer address validation
	return nil
}

// TODO(srene): implement DRS/height verification
func (v *StateUpdateValidator) validateDRS(startHeight, endHeight uint64) error {
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
