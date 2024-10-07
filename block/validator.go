package block

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/settlement"
	"github.com/dymensionxyz/dymint/types"
)

// SyncTargetLoop listens for syncing events (from new state update or from initial syncing) and syncs to the last submitted height.
// In case the node is already synced, it validate
func (m *Manager) ValidateLoop(ctx context.Context) error {

	for {
		select {
		case <-ctx.Done():
			return nil
		case stateIndex := <-m.validateC:

			// Validate new state update
			batch, err := m.SLClient.GetBatchAtIndex(stateIndex)

			if err != nil {
				return err
			}
			err = m.validator.ValidateStateUpdate(batch)
			if err != nil {
				m.logger.Error("state update validation", "error", err)
			}
		}
	}
}

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

var _ DABlocksValidator = &StateUpdateValidator{}
var _ P2PBlockValidator = &StateUpdateValidator{}

// NewValidator creates a new Validator.
func NewStateUpdateValidator(logger types.Logger, blockManager *Manager) *StateUpdateValidator {
	return &StateUpdateValidator{
		logger:       logger,
		blockManager: blockManager,
	}
}

func (v *StateUpdateValidator) ValidateStateUpdate(batch *settlement.ResultRetrieveBatch) error {

	err := v.validateDRS(batch.StartHeight, batch.EndHeight, batch.DRSVersion)
	if err != nil {
		return err
	}

	var daBlocks []*types.Block
	var p2pBlocks []*types.Block

	//load blocks for the batch height, either P2P or DA blocks
	for height := batch.StartHeight; height <= batch.EndHeight; height++ {
		source, err := v.blockManager.Store.LoadBlockSource(height)
		if err != nil {
			return err
		}
		block, err := v.blockManager.Store.LoadBlock(height)
		if err != nil {
			return err
		}
		if source == types.DA.String() {
			daBlocks = append(daBlocks, block)
		} else {
			p2pBlocks = append(p2pBlocks, block)
		}
	}

	numBlocks := batch.EndHeight - batch.StartHeight + 1
	if uint64(len(daBlocks)) != numBlocks {
		daBatch := v.blockManager.Retriever.RetrieveBatches(batch.MetaData.DA)
		if daBatch.Code != da.StatusSuccess {
			return daBatch.Error
		}

		for _, batch := range daBatch.Batches {
			daBlocks = append(daBlocks, batch.Blocks...)
		}
	}

	err = v.ValidateDaBlocks(batch, daBlocks)
	if err != nil {
		return err
	}

	if len(p2pBlocks) > 0 {
		err = v.ValidateP2PBlocks(daBlocks, p2pBlocks)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *StateUpdateValidator) ValidateP2PBlocks(daBlocks []*types.Block, p2pBlocks []*types.Block) error {

	i := 0
	for _, daBlock := range daBlocks {

		if p2pBlocks[i].Header.Height != daBlocks[i].Header.Height {
			break
		}
		p2pBlockHash, err := blockHash(p2pBlocks[i])
		if err != nil {
			return err
		}
		i++
		daBlockHash, err := blockHash(daBlock)
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
		return nil, fmt.Errorf("error hashing block. err: %w", err)
	}
	h := sha256.New()
	h.Write(blockBytes)
	return h.Sum(nil), nil
}
