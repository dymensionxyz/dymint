package store

import "fmt"

// PruneBlocks removes block up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneBlocks(heightInt int64) (uint64, error) {
	if heightInt <= 0 {
		return 0, fmt.Errorf("height must be greater than 0")
	}

	height := uint64(heightInt)
	if height > s.Height() {
		return 0, fmt.Errorf("cannot prune beyond the latest height %v", s.height)
	}
	base := s.Base()
	if height < base {
		return 0, fmt.Errorf("cannot prune to height %v, it is lower than base height %v",
			height, base)
	}

	pruned := uint64(0)
	batch := s.db.NewBatch()
	defer batch.Discard()

	flush := func(batch Batch, base uint64) error {
		err := batch.Commit()
		if err != nil {
			return fmt.Errorf("prune up to height %v: %w", base, err)
		}
		s.SetBase(base)
		return nil
	}

	for h := base; h < height; h++ {
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
			batch = s.db.NewBatch()
			defer batch.Discard()
		}
	}

	err := flush(batch, height)
	if err != nil {
		return 0, err
	}
	return pruned, nil
}
