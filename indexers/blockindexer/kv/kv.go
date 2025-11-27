package kv

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/google/orderedcode"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"

	indexer "github.com/dymensionxyz/dymint/indexers/blockindexer"
	"github.com/dymensionxyz/dymint/store"

	tmtypes "github.com/tendermint/tendermint/types"

	dmtypes "github.com/dymensionxyz/dymint/types/pb/dymint"
)

var _ indexer.BlockIndexer = (*BlockerIndexer)(nil)

// BlockerIndexer implements a block indexer, indexing BeginBlock and EndBlock
// events with an underlying KV store. Block events are indexed by their height,
// such that matching search criteria returns the respective block height(s).
type BlockerIndexer struct {
	store store.KV
}

func New(store store.KV) *BlockerIndexer {
	return &BlockerIndexer{
		store: store,
	}
}

// Has returns true if the given height has been indexed. An error is returned
// upon database query failure.
func (idx *BlockerIndexer) Has(height int64) (bool, error) {
	key, err := heightKey(height)
	if err != nil {
		return false, fmt.Errorf("create block height index key: %w", err)
	}

	_, err = idx.store.Get(key)
	if errors.Is(err, gerrc.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Index indexes BeginBlock and EndBlock events for a given block by its height.
// The following is indexed:
//
// primary key: encode(block.height | height) => encode(height)
// BeginBlock events: encode(eventType.eventAttr|eventValue|height|begin_block) => encode(height)
// EndBlock events: encode(eventType.eventAttr|eventValue|height|end_block) => encode(height)
func (idx *BlockerIndexer) Index(bh tmtypes.EventDataNewBlockHeader) error {
	batch := idx.store.NewBatch()
	defer batch.Discard()

	height := bh.Header.Height
	// 1. index by height
	key, err := heightKey(height)
	if err != nil {
		return fmt.Errorf("create block height index key: %w", err)
	}
	if err := batch.Set(key, int64ToBytes(height)); err != nil {
		return err
	}

	// 2. index BeginBlock events
	beginKeys, err := idx.indexEvents(batch, bh.ResultBeginBlock.Events, "begin_block", height)
	if err != nil {
		return fmt.Errorf("index BeginBlock events: %w", err)
	}
	// 3. index EndBlock events
	endKeys, err := idx.indexEvents(batch, bh.ResultEndBlock.Events, "end_block", height)
	if err != nil {
		return fmt.Errorf("index EndBlock events: %w", err)
	}

	// 4. index all eventkeys by height key for easy pruning
	err = idx.addEventKeys(height, &beginKeys, &endKeys, batch)
	if err != nil {
		return err
	}
	return batch.Commit()
}

// Search performs a query for block heights that match a given BeginBlock
// and Endblock event search criteria. The given query can match against zero,
// one or more block heights. In the case of height queries, i.e. block.height=H,
// if the height is indexed, that height alone will be returned. An error and
// nil slice is returned. Otherwise, a non-nil slice and nil error is returned.
func (idx *BlockerIndexer) Search(ctx context.Context, q *query.Query) ([]int64, error) {
	results := make([]int64, 0)
	select {
	case <-ctx.Done():
		return results, nil

	default:
	}

	conditions, err := q.Conditions()
	if err != nil {
		return nil, fmt.Errorf("parse query conditions: %w", err)
	}

	// If there is an exact height query, return the result immediately
	// (if it exists).
	height, ok := lookForHeight(conditions)
	if ok {
		ok, err := idx.Has(height)
		if err != nil {
			return nil, err
		}

		if ok {
			return []int64{height}, nil
		}

		return results, nil
	}

	var heightsInitialized bool
	filteredHeights := make(map[string][]byte)

	// conditions to skip because they're handled before "everything else"
	skipIndexes := make([]int, 0)

	// Extract ranges. If both upper and lower bounds exist, it's better to get
	// them in order as to not iterate over kvs that are not within range.
	ranges, rangeIndexes := indexer.LookForRanges(conditions)
	if len(ranges) > 0 {
		skipIndexes = append(skipIndexes, rangeIndexes...)

		for _, qr := range ranges {
			prefix, err := orderedcode.Append(nil, qr.Key)
			if err != nil {
				return nil, fmt.Errorf("create prefix key: %w", err)
			}

			if !heightsInitialized {
				filteredHeights, err = idx.matchRange(ctx, qr, prefix, filteredHeights, true)
				if err != nil {
					return nil, err
				}

				heightsInitialized = true

				// Ignore any remaining conditions if the first condition resulted in no
				// matches (assuming implicit AND operand).
				if len(filteredHeights) == 0 {
					break
				}
			} else {
				filteredHeights, err = idx.matchRange(ctx, qr, prefix, filteredHeights, false)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// for all other conditions
	for i, c := range conditions {
		if intInSlice(i, skipIndexes) {
			continue
		}

		startKey, err := orderedcode.Append(nil, c.CompositeKey, fmt.Sprintf("%v", c.Operand))
		if err != nil {
			return nil, err
		}

		if !heightsInitialized {
			filteredHeights, err = idx.match(ctx, c, startKey, filteredHeights, true)
			if err != nil {
				return nil, err
			}

			heightsInitialized = true

			// Ignore any remaining conditions if the first condition resulted in no
			// matches (assuming implicit AND operand).
			if len(filteredHeights) == 0 {
				break
			}
		} else {
			filteredHeights, err = idx.match(ctx, c, startKey, filteredHeights, false)
			if err != nil {
				return nil, err
			}
		}
	}

	// fetch matching heights
	results = make([]int64, 0, len(filteredHeights))
	for _, hBz := range filteredHeights {
		cont := true

		h := int64FromBytes(hBz)

		ok, err := idx.Has(h)
		if err != nil {
			return nil, err
		}
		if ok {
			results = append(results, h)
		}

		select {
		case <-ctx.Done():
			cont = false

		default:
		}
		if !cont {
			break
		}
	}

	slices.Sort(results)

	return results, nil
}

// matchRange returns all matching block heights that match a given QueryRange
// and start key. An already filtered result (filteredHeights) is provided such
// that any non-intersecting matches are removed.
//
// NOTE: The provided filteredHeights may be empty if no previous condition has
// matched.
func (idx *BlockerIndexer) matchRange(
	ctx context.Context,
	qr indexer.QueryRange,
	startKey []byte,
	filteredHeights map[string][]byte,
	firstRun bool,
) (map[string][]byte, error) {
	// A previous match was attempted but resulted in no matches, so we return
	// no matches (assuming AND operand).
	if !firstRun && len(filteredHeights) == 0 {
		return filteredHeights, nil
	}

	tmpHeights := make(map[string][]byte)
	lowerBound := qr.LowerBoundValue()
	upperBound := qr.UpperBoundValue()

	it := idx.store.PrefixIterator(startKey)
	defer it.Discard()

LOOP:
	for ; it.Valid(); it.Next() {
		cont := true

		var (
			eventValue string
			err        error
		)

		if qr.Key == tmtypes.BlockHeightKey {
			eventValue, err = parseValueFromPrimaryKey(it.Key())
		} else {
			eventValue, err = parseValueFromEventKey(it.Key())
		}

		if err != nil {
			continue
		}

		if _, ok := qr.AnyBound().(int64); ok {
			v, err := strconv.ParseInt(eventValue, 10, 64)
			if err != nil {
				continue LOOP
			}

			include := true
			if lowerBound != nil && v < lowerBound.(int64) {
				include = false
			}

			if upperBound != nil && v > upperBound.(int64) {
				include = false
			}

			if include {
				tmpHeights[string(it.Value())] = it.Value()
			}
		}

		select {
		case <-ctx.Done():
			cont = false

		default:
		}

		if !cont {
			break
		}
	}

	if err := it.Error(); err != nil {
		return nil, err
	}

	if len(tmpHeights) == 0 || firstRun {
		// Either:
		//
		// 1. Regardless if a previous match was attempted, which may have had
		// results, but no match was found for the current condition, then we
		// return no matches (assuming AND operand).
		//
		// 2. A previous match was not attempted, so we return all results.
		return tmpHeights, nil
	}

	// Remove/reduce matches in filteredHashes that were not found in this
	// match (tmpHashes).
	for k := range filteredHeights {
		cont := true

		if tmpHeights[k] == nil {
			delete(filteredHeights, k)

			select {
			case <-ctx.Done():
				cont = false

			default:
			}
		}

		if !cont {
			break
		}
	}

	return filteredHeights, nil
}

// match returns all matching heights that meet a given query condition and start
// key. An already filtered result (filteredHeights) is provided such that any
// non-intersecting matches are removed.
//
// NOTE: The provided filteredHeights may be empty if no previous condition has
// matched.
func (idx *BlockerIndexer) match(
	ctx context.Context,
	c query.Condition,
	startKeyBz []byte,
	filteredHeights map[string][]byte,
	firstRun bool,
) (map[string][]byte, error) {
	// A previous match was attempted but resulted in no matches, so we return
	// no matches (assuming AND operand).
	if !firstRun && len(filteredHeights) == 0 {
		return filteredHeights, nil
	}

	tmpHeights := make(map[string][]byte)

	switch c.Op {
	case query.OpEqual:
		it := idx.store.PrefixIterator(startKeyBz)
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			tmpHeights[string(it.Value())] = it.Value()

			if err := ctx.Err(); err != nil {
				break
			}
		}

		if err := it.Error(); err != nil {
			return nil, err
		}

	case query.OpExists:
		prefix, err := orderedcode.Append(nil, c.CompositeKey)
		if err != nil {
			return nil, err
		}

		it := idx.store.PrefixIterator(prefix)
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			tmpHeights[string(it.Value())] = it.Value()

			select {
			case <-ctx.Done():
				cont = false

			default:
			}

			if !cont {
				break
			}
		}

		if err := it.Error(); err != nil {
			return nil, err
		}

	case query.OpContains:
		prefix, err := orderedcode.Append(nil, c.CompositeKey)
		if err != nil {
			return nil, err
		}

		it := idx.store.PrefixIterator(prefix)
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			eventValue, err := parseValueFromEventKey(it.Key())
			if err != nil {
				continue
			}

			if strings.Contains(eventValue, c.Operand.(string)) {
				tmpHeights[string(it.Value())] = it.Value()
			}

			select {
			case <-ctx.Done():
				cont = false

			default:
			}

			if !cont {
				break
			}
		}
		if err := it.Error(); err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("other operators should be handled already")
	}

	if len(tmpHeights) == 0 || firstRun {
		// Either:
		//
		// 1. Regardless if a previous match was attempted, which may have had
		// results, but no match was found for the current condition, then we
		// return no matches (assuming AND operand).
		//
		// 2. A previous match was not attempted, so we return all results.
		return tmpHeights, nil
	}

	// Remove/reduce matches in filteredHeights that were not found in this
	// match (tmpHeights).
	for k := range filteredHeights {
		cont := true

		if tmpHeights[k] == nil {
			delete(filteredHeights, k)

			select {
			case <-ctx.Done():
				cont = false

			default:
			}
		}

		if !cont {
			break
		}
	}

	return filteredHeights, nil
}

func (idx *BlockerIndexer) indexEvents(batch store.KVBatch, events []abci.Event, typ string, height int64) (dmtypes.EventKeys, error) {
	heightBz := int64ToBytes(height)
	keys := dmtypes.EventKeys{}
	for _, event := range events {
		// only index events with a non-empty type
		if len(event.Type) == 0 {
			continue
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

			// index iff the event specified index:true and it's not a reserved event
			compositeKey := fmt.Sprintf("%s.%s", event.Type, string(attr.Key))
			if compositeKey == tmtypes.BlockHeightKey {
				return dmtypes.EventKeys{}, fmt.Errorf("event type and attribute key \"%s\" is reserved; please use a different key", compositeKey)
			}

			if attr.GetIndex() {
				key, err := eventKey(compositeKey, typ, string(attr.Value), height)
				if err != nil {
					return dmtypes.EventKeys{}, fmt.Errorf("create block index key: %w", err)
				}

				if err := batch.Set(key, heightBz); err != nil {
					return dmtypes.EventKeys{}, err
				}
				keys.Keys = append(keys.Keys, key)
			}
		}
	}

	return keys, nil
}

func (idx *BlockerIndexer) Prune(from, to uint64, logger log.Logger) (uint64, error) {
	return idx.pruneBlocks(from, to, logger)
}

func (idx *BlockerIndexer) pruneBlocks(from, to uint64, logger log.Logger) (uint64, error) {
	pruned := uint64(0)
	toFlush := uint64(0)
	batch := idx.store.NewBatch()
	defer batch.Discard()

	flush := func(batch store.KVBatch, height int64) error {
		err := batch.Commit()
		if err != nil {
			return fmt.Errorf("flush batch to disk: height %d: %w", height, err)
		}
		return nil
	}

	for h := int64(from); h < int64(to); h++ { //nolint:gosec // heights (from and to) are always positive and fall in int64

		// flush every 1000 blocks to avoid batches becoming too large
		if toFlush > 1000 {
			err := flush(batch, h)
			if err != nil {
				return 0, err
			}
			batch.Discard()
			batch = idx.store.NewBatch()

			toFlush = 0
		}

		ok, err := idx.Has(h)
		if err != nil {
			logger.Debug("pruning block indexer checking height", "height", h, "err", err)
			continue
		}
		if !ok {
			continue
		}

		key, err := heightKey(h)
		if err != nil {
			logger.Debug("pruning block indexer getting height key", "height", h, "err", err)
			continue
		}
		if err := batch.Delete(key); err != nil {
			logger.Debug("pruning block indexer deleting height key", "height", h, "err", err)
			continue
		}

		pruned++
		toFlush++

		prunedEvents, err := idx.pruneEvents(h, logger, batch)
		if err != nil {
			logger.Debug("pruning block indexer events", "height", h, "err", err)
			continue
		}
		pruned += prunedEvents
		toFlush += prunedEvents

	}

	err := flush(batch, int64(to)) //nolint:gosec // height is non-negative and falls in int64
	if err != nil {
		return 0, err
	}

	return pruned, nil
}

func (idx *BlockerIndexer) pruneEvents(height int64, logger log.Logger, batch store.KVBatch) (uint64, error) {
	pruned := uint64(0)

	eventKey, err := eventHeightKey(height)
	if err != nil {
		return pruned, err
	}
	keysList, err := idx.store.Get(eventKey)
	if err != nil {
		return pruned, err
	}
	eventKeys := &dmtypes.EventKeys{}
	err = eventKeys.Unmarshal(keysList)
	if err != nil {
		return pruned, err
	}
	for _, key := range eventKeys.Keys {
		pruned++
		err := batch.Delete(key)
		if err != nil {
			logger.Error("pruning block indexer iterate events", "height", height, "err", err)
			continue
		}

	}
	return pruned, nil
}

func (idx *BlockerIndexer) addEventKeys(height int64, beginKeys *dmtypes.EventKeys, endKeys *dmtypes.EventKeys, batch store.KVBatch) error {
	eventKeys := beginKeys
	eventKeys.Keys = append(eventKeys.Keys, endKeys.Keys...)
	eventKeyHeight, err := eventHeightKey(height)
	if err != nil {
		return err
	}
	eventKeysBytes, err := eventKeys.Marshal()
	if err != nil {
		return err
	}
	if err := batch.Set(eventKeyHeight, eventKeysBytes); err != nil {
		return err
	}
	return nil
}
