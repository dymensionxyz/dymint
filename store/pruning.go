package store

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// PruneStore removes blocks up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneStore(to uint64, logger types.Logger) (uint64, error) {
	pruned := uint64(0)
	from, err := s.LoadBaseHeight()
	if errors.Is(err, gerrc.ErrNotFound) {
		logger.Error("load store base height", "err", err)
	} else if err != nil {
		return pruned, err
	}
	pruned, err = s.PruneHeights(from, to, logger)
	if err != nil {
		return pruned, fmt.Errorf("pruning blocks. from: %d to: %d: err:%w", from, to, err)
	}

	err = s.SaveBaseHeight(to)
	if err != nil {
		logger.Error("saving base height", "error", err)
	}
	return pruned, nil
}

// pruneHeights prunes all store entries that are stored along blocks (blocks,commit,proposer, etc)
func (s *DefaultStore) PruneHeights(from, to uint64, logger types.Logger) (uint64, error) {
	pruneBlocks := func(batch KVBatch, height uint64) error {
		hash, err := s.loadHashFromIndex(height)
		if err != nil {
			return err
		}
		if err := batch.Delete(getBlockKey(hash)); err != nil {
			logger.Error("delete block", "error", err)
		}
		if err := batch.Delete(getCommitKey(hash)); err != nil {
			logger.Error("delete commit", "error", err)
		}

		if err := batch.Delete(getIndexKey(height)); err != nil {
			logger.Error("delete hash index", "error", err)
		}

		if err := batch.Delete(getResponsesKey(height)); err != nil {
			logger.Error("delete responses", "error", err)
		}
		if err := batch.Delete(getDRSVersionKey(height)); err != nil {
			logger.Error("delete drs", "error", err)
		}
		if err := batch.Delete(getProposerKey(height)); err != nil {
			logger.Error("delete proposer", "error", err)
		}

		return nil
	}

	pruned, err := s.prune(from, to, pruneBlocks, logger)
	return pruned, err
}

// prune is the function  that iterates through all heights and prunes according to the pruning function set, doesn't include 'to'
func (s *DefaultStore) prune(from, to uint64, prune func(batch KVBatch, height uint64) error, logger types.Logger) (uint64, error) {
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
			logger.Debug("unable to prune", "height", h, "err", err)
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
