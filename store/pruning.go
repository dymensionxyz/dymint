package store

import (
	"fmt"

	"github.com/dymensionxyz/dymint/gerr"
)

// PruneBlocks removes block up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneBlocks(from, to uint64) (uint64, error) {
	if from <= 0 {
		return 0, fmt.Errorf("from height must be greater than 0: %w", gerr.ErrInvalidArgument)
	}

	if to <= from {
		return 0, fmt.Errorf("to height must be greater than from height: to: %d: from: %d: %w", to, from, gerr.ErrInvalidArgument)
	}

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
		hash, err := s.loadHashFromIndex(h)
		if err != nil {
			continue
		}
		if err := batch.Delete(getBlockKey(hash)); err != nil {
			return 0, err
		}
		if err := batch.Delete(getCommitKey(hash)); err != nil {
			return 0, err
		}
		if err := batch.Delete(getIndexKey(h)); err != nil {
			return 0, err
		}
		if err := batch.Delete(getResponsesKey(h)); err != nil {
			return 0, err
		}
		if err := batch.Delete(getValidatorsKey(h)); err != nil {
			return 0, err
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
