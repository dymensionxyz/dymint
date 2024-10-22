package store

import (
	"fmt"

	"github.com/dymensionxyz/dymint/types"
)

// PruneStore removes blocks up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneStore(to uint64, logger types.Logger) (uint64, error) {
	from, err := s.LoadBaseHeight()
	if err != nil {
		return uint64(0), fmt.Errorf("unable to retrieve stored base: %w", err)
	}
	prunedBlocks, err := s.pruneBlocks(from, to, logger)
	if err != nil {
		logger.Error("pruning blocks", "from", from, "to", to, "blocks pruned", prunedBlocks, "err", err)
	}

	prunedResponses, err := s.pruneResponses(from, to, logger)
	if err != nil {
		logger.Error("pruning responses", "from", from, "to", to, "responses pruned", prunedResponses, "err", err)
	}

	err = s.SaveBaseHeight(from + prunedBlocks)
	if err != nil {
		logger.Error("saving base height", "error", err)
	}
	return prunedBlocks, nil
}

// pruneBlocks prunes all store entries that are stored along blocks (blocks,commit and block hash)
func (s *DefaultStore) pruneBlocks(from, to uint64, logger types.Logger) (uint64, error) {
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

	prunedBlocks, err := s.pruneHeights(from, to, pruneBlocks, logger)
	return prunedBlocks, err
}

// pruneResponses prunes block execution responses from store
func (s *DefaultStore) pruneResponses(from, to uint64, logger types.Logger) (uint64, error) {
	pruneResponses := func(batch KVBatch, height uint64) error {
		return batch.Delete(getResponsesKey(height))
	}

	prunedResponses, err := s.pruneHeights(from, to, pruneResponses, logger)
	return prunedResponses, err
}

// pruneHeights is the common function for all pruning that iterates through all heights and prunes according to the pruning function set
func (s *DefaultStore) pruneHeights(from, to uint64, prune func(batch KVBatch, height uint64) error, logger types.Logger) (uint64, error) {
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
		if err := batch.Delete(getCidKey(h)); err != nil {
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
