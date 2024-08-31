package store

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// PruneBlocks removes block up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneBlocks(from, to uint64, logger types.Logger) (uint64, error) {
	if from <= 0 {
		return 0, fmt.Errorf("from height must be greater than 0: %w", gerrc.ErrInvalidArgument)
	}

	if to <= from {
		return 0, fmt.Errorf("to height must be greater than from height: to: %d: from: %d: %w", to, from, gerrc.ErrInvalidArgument)
	}

	pruneBlocks := func(batch KVBatch, height uint64) error {
		hash, err := s.loadHashFromIndex(height)
		if err != nil {
			return err
		}
		if err := batch.Delete(getBlockKey(hash)); err != nil {
			return err
		}
		if err := batch.Delete(getCommitKey(hash)); err != nil {
			return err
		}
		if err := batch.Delete(getIndexKey(height)); err != nil {
			return err
		}
		return nil
	}

	pruneResponses := func(batch KVBatch, height uint64) error {
		if err := batch.Delete(getResponsesKey(height)); err != nil {
			return err
		}
		return nil
	}

	pruneSequencers := func(batch KVBatch, height uint64) error {
		if err := batch.Delete(getSequencersKey(height)); err != nil {
			return err
		}
		return nil
	}

	pruneCids := func(batch KVBatch, height uint64) error {
		if err := batch.Delete(getCidKey(height)); err != nil {
			return err
		}
		return nil
	}

	prunedBlocks, err := s.pruneIteration(from, to, pruneBlocks)
	if err != nil {
		logger.Error("pruning blocks", "from", from, "to", to, "err", err)
	}

	_, err = s.pruneIteration(from, to, pruneResponses)
	if err != nil {
		logger.Error("pruning responses", "from", from, "to", to, "err", err)
	}

	_, err = s.pruneIteration(from, to, pruneSequencers)
	if err != nil {
		logger.Error("pruning sequencers", "from", from, "to", to, "err", err)
	}

	_, err = s.pruneIteration(from, to, pruneCids)
	if err != nil {
		logger.Error("pruning cids", "from", from, "to", to, "err", err)
	}
	return prunedBlocks, nil

}

func (s *DefaultStore) pruneIteration(from, to uint64, prune func(batch KVBatch, height uint64) error) (uint64, error) {

	pruned := uint64(0)

	batch := s.db.NewBatch()
	defer batch.Discard()

	flush := func(batch KVBatch, height uint64) error {
		err := batch.Commit()
		if err != nil {
			return fmt.Errorf("flush batch to disk: height %d: %w", height, err)
		}
		return nil
	}

	for h := from; h < to; h++ {
		err := prune(batch, h)
		if err != nil {
			continue
		}
		pruned++

		// flush every 1000 blocks to avoid batches becoming too large
		if pruned%1000 == 0 && pruned > 0 {
			err := flush(batch, h)
			if err != nil {
				return 0, err
			}
			batch.Discard()
			batch = s.db.NewBatch()
		}

	}
	err := flush(batch, to)
	if err != nil {
		return 0, err
	}

	return pruned, nil

}
