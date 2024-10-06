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
	ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) (da.ResultRetrieveBatch, error)
}

// StateUpdateValidator is a validator for messages gossiped in the p2p network.
type StateUpdateValidator struct {
	logger    types.Logger
	retriever da.BatchRetriever
}

var _ DABlocksValidator = &StateUpdateValidator{}
var _ P2PBlockValidator = &StateUpdateValidator{}

// NewValidator creates a new Validator.
func NewStateUpdateValidator(logger types.Logger, daClient da.BatchRetriever) *StateUpdateValidator {
	return &StateUpdateValidator{
		logger:    logger,
		retriever: daClient,
	}
}

func (v *StateUpdateValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) (da.ResultRetrieveBatch, error) {

	//TODO(srene): check if pending validations because applied blocks from cache.

	err := v.validateDRS(batch.StartHeight, batch.EndHeight, batch.DRSVersion)
	if err != nil {
		return da.ResultRetrieveBatch{}, err
	}
	daBatch := v.retriever.RetrieveBatches(batch.MetaData.DA)
	if daBatch.Code != da.StatusSuccess {
		return da.ResultRetrieveBatch{}, daBatch.Error
	}

	var daBlocks []*types.Block
	for _, batch := range daBatch.Batches {
		daBlocks = append(daBlocks, batch.Blocks...)
	}

	err = v.validateDaBlocks(batch, daBlocks)
	if err != nil {
		return da.ResultRetrieveBatch{}, err
	}

	return daBatch, nil
}

func (v *StateUpdateValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks []*types.Block) error {
	if len(daBlocks) != len(p2pBlocks) {
		return fmt.Errorf("comparing different number of blocks between P2P blocks and DA blocks. Start height: %d End Height: %d", daBlocks[0].Header.Height, daBlocks[len(daBlocks)-1].Header.Height)
	}

	for i, p2pBlock := range daBlocks {

		p2pBlockHash, err := blockHash(p2pBlock)
		if err != nil {
			return err
		}
		daBlockHash, err := blockHash(daBlocks[i])
		if err != nil {
			return err
		}
		if !bytes.Equal(p2pBlockHash, daBlockHash) {
			return fmt.Errorf("failed comparing blocks")
		}
	}
	return nil
}

// isHeightFinalized is required to skip old blocks validation
func (v *StateUpdateValidator) isHeightFinalized(height uint64) bool {
	return false
}

func (v *StateUpdateValidator) validateDaBlocks(slBatch *settlement.ResultRetrieveBatch, daBlocks []*types.Block) error {

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
		if bd.Timestamp != daBlocks[i].Header.GetTimestamp() {
			return fmt.Errorf("timestamp mismatch between state update and DA batch. State index: %d: Height: %d Timestamp SL: %s Timestamp DA: %s", slBatch.StateIndex, bd.Height, bd.Timestamp, daBlocks[i].Header.GetTimestamp())
		}
	}

	//TODO(srene): implement sequencer address validation
	return nil
}

func (v *StateUpdateValidator) validateDRS(startHeight, endHeight uint64, version string) error {

	//TODO(srene): implement DRS/height verification
	return nil
}

func blockHash(block *types.Block) ([]byte, error) {
	blockBytes, err := block.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("error marshal p2pblock. err: %w", err)
	}
	h := sha256.New()
	h.Write(blockBytes)
	return h.Sum(nil), nil
}
