package block

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

type P2PBlockValidator interface {
	ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks []*types.Block) error
}

type DABlocksValidator interface {
	ValidateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error
}

// StateUpdateValidator is a validator for messages gossiped in the p2p network.
type StateUpdateValidator struct {
	logger       types.Logger
	blockManager *Manager
}

var (
	_ DABlocksValidator = &StateUpdateValidator{}
	_ P2PBlockValidator = &StateUpdateValidator{}
)

// NewValidator creates a new Validator.
func NewStateUpdateValidator(logger types.Logger, blockManager *Manager) *StateUpdateValidator {
	return &StateUpdateValidator{
		logger:       logger,
		blockManager: blockManager,
	}
}

func (v *StateUpdateValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {
	v.logger.Debug("validating state update", "start height", batch.StartHeight, "end height", batch.EndHeight)

	err := v.validateDRS(batch.StartHeight, batch.EndHeight, batch.DRSVersion)
	if err != nil {
		return err
	}

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

	// if not all blocks are applied from DA, it is necessary to get all batch blocks from DA
	numBlocks := batch.EndHeight - batch.StartHeight + 1
	if uint64(len(daBlocks)) != numBlocks {
		daBlocks = []*types.Block{}
		daBatch := v.blockManager.Retriever.RetrieveBatches(batch.MetaData.DA)
		if daBatch.Code != da.StatusSuccess {
			return daBatch.Error
		}
		for _, batch := range daBatch.Batches {
			daBlocks = append(daBlocks, batch.Blocks...)
		}
	}

	// validate DA blocks against the state update
	err = v.ValidateDaBlocks(batch, daBlocks)
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
			return fmt.Errorf("p2p block different from DA block. p2p height: %d, DA height: %d", p2pBlocks[i].Header.Height, daBlock.Header.Height)
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
	numSlBlocks := len(slBatch.BlockDescriptors)
	numDABlocks := len(daBlocks)
	if numSlBlocks != numDABlocks {
		return fmt.Errorf("num blocks mismatch between state update and DA batch. State index: %d State update blocks: %d DA batch blocks: %d", slBatch.StateIndex, numSlBlocks, numDABlocks)
	}

	// check blocks
	for i, bd := range slBatch.BlockDescriptors {
		// height check
		if bd.Height != daBlocks[i].Header.Height {
			return fmt.Errorf("height mismatch between state update and DA batch. State index: %d SL height: %d DA height: %d", slBatch.StateIndex, bd.Height, daBlocks[i].Header.Height)
		}
		// we compare the state root between SL state info and DA block
		if !bytes.Equal(bd.StateRoot, daBlocks[i].Header.AppHash[:]) {
			return fmt.Errorf("state root mismatch between state update and DA batch. State index: %d: Height: %d State root SL: %d State root DA: %d", slBatch.StateIndex, bd.Height, bd.StateRoot, daBlocks[i].Header.AppHash[:])
		}

		// we compare the timestamp between SL state info and DA block
		if !bd.Timestamp.Equal(daBlocks[i].Header.GetTimestamp()) {
			return fmt.Errorf("timestamp mismatch between state update and DA batch. State index: %d: Height: %d Timestamp SL: %s Timestamp DA: %s", slBatch.StateIndex, bd.Height, bd.Timestamp.UTC(), daBlocks[i].Header.GetTimestamp().UTC())
		}
	}

	// TODO(srene): implement sequencer address validation
	return nil
}

// TODO(srene): implement DRS/height verification
func (v *StateUpdateValidator) validateDRS(startHeight, endHeight uint64, version string) error {
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
