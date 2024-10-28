package store

import (
	"errors"
	"fmt"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
)

// PruneStore removes blocks up to (but not including) a height. It returns number of blocks pruned.
func (s *DefaultStore) PruneStore(to uint64, logger types.Logger) (uint64, error) {
	prunedBlocks := uint64(0)
	from, err := s.LoadBaseHeight()
	if errors.Is(err, gerrc.ErrNotFound) {
		logger.Error("load store base height", "err", err)
	} else if err != nil {
		return prunedBlocks, err
	}
	prunedBlocks, err = s.pruneBlocks(from, to, logger)
	if err != nil {
		logger.Error("pruning blocks", "from", from, "to", to, "blocks pruned", prunedBlocks, "err", err)
	}

	prunedResponses, err := s.pruneResponses(from, to, logger)
	if err != nil {
		logger.Error("pruning responses", "from", from, "to", to, "responses pruned", prunedResponses, "err", err)
	}

	prunedDRS, err := s.pruneDRSVersion(from, to, logger)
	if err != nil {
		logger.Error("pruning drs version", "from", from, "to", to, "drs pruned", prunedDRS, "err", err)
	}

	prunedProposers, err := s.pruneProposers(from, to, logger)
	if err != nil {
		logger.Error("pruning block sync identifiers", "from", from, "to", to, "proposers pruned", prunedProposers, "err", err)
	prunedSequencers, err := s.pruneSequencers(from, to, logger)
	if err != nil {
		logger.Error("pruning sequencers", "from", from, "to", to, "sequencers pruned", prunedSequencers, "err", err)
	}

	err = s.SaveBaseHeight(to)
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

// pruneDRSVersion prunes drs version info from store
func (s *DefaultStore) pruneDRSVersion(from, to uint64, logger types.Logger) (uint64, error) {
	pruneDRS := func(batch KVBatch, height uint64) error {
		return batch.Delete(getDRSVersionKey(height))
	}
	prunedSequencers, err := s.pruneHeights(from, to, pruneDRS, logger)
}

// pruneSequencers prunes sequencers from store
func (s *DefaultStore) pruneSequencers(from, to uint64, logger types.Logger) (uint64, error) {
	pruneSequencers := func(batch KVBatch, height uint64) error {
		return batch.Delete(getSequencersKey(height))
	}
	prunedSequencers, err := s.pruneHeights(from, to, pruneSequencers, logger)
	return prunedSequencers, err
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

// pruneSequencers prunes proposer from store
func (s *DefaultStore) pruneProposers(from, to uint64, logger types.Logger) (uint64, error) {
	pruneProposers := func(batch KVBatch, height uint64) error {
		return batch.Delete(getProposerKey(height))
	}
	prunedSequencers, err := s.pruneHeights(from, to, pruneProposers, logger)
	return prunedSequencers, err
}
