package kv

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

	"github.com/dymensionxyz/dymint/types/pb/dymint"
	dmtypes "github.com/dymensionxyz/dymint/types/pb/dymint"
)

var _ indexer.BlockIndexer = (*BlockerIndexer)(nil)

type BlockerIndexer struct {
	store store.KV
}

func New(store store.KV) *BlockerIndexer {
	return &BlockerIndexer{
		store: store,
	}
}

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

func (idx *BlockerIndexer) Index(bh tmtypes.EventDataNewBlockHeader) error {
	batch := idx.store.NewBatch()
	defer batch.Discard()

	height := bh.Header.Height

	key, err := heightKey(height)
	if err != nil {
		return fmt.Errorf("create block height index key: %w", err)
	}
	if err := batch.Set(key, int64ToBytes(height)); err != nil {
		return err
	}

	beginKeys, err := idx.indexEvents(batch, bh.ResultBeginBlock.Events, "begin_block", height)
	if err != nil {
		return fmt.Errorf("index BeginBlock events: %w", err)
	}

	endKeys, err := idx.indexEvents(batch, bh.ResultEndBlock.Events, "end_block", height)
	if err != nil {
		return fmt.Errorf("index EndBlock events: %w", err)
	}

	err = idx.addEventKeys(height, &beginKeys, &endKeys, batch)
	if err != nil {
		return err
	}
	return batch.Commit()
}

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

	skipIndexes := make([]int, 0)

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

	sort.Slice(results, func(i, j int) bool { return results[i] < results[j] })

	return results, nil
}

func (idx *BlockerIndexer) matchRange(
	ctx context.Context,
	qr indexer.QueryRange,
	startKey []byte,
	filteredHeights map[string][]byte,
	firstRun bool,
) (map[string][]byte, error) {
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
		return tmpHeights, nil
	}

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

func (idx *BlockerIndexer) match(
	ctx context.Context,
	c query.Condition,
	startKeyBz []byte,
	filteredHeights map[string][]byte,
	firstRun bool,
) (map[string][]byte, error) {
	if !firstRun && len(filteredHeights) == 0 {
		return filteredHeights, nil
	}

	tmpHeights := make(map[string][]byte)

	switch {
	case c.Op == query.OpEqual:
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

	case c.Op == query.OpExists:
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

	case c.Op == query.OpContains:
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
		return tmpHeights, nil
	}

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

		if len(event.Type) == 0 {
			continue
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

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

	for h := int64(from); h < int64(to); h++ {

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
			continue
		}
		if !ok {
			continue
		}

		key, err := heightKey(h)
		if err != nil {
			continue
		}
		if err := batch.Delete(key); err != nil {
			continue
		}

		pruned++
		toFlush++

		prunedEvents, err := idx.pruneEvents(h, logger, batch)
		if err != nil {
			continue
		}
		pruned += prunedEvents
		toFlush += prunedEvents

	}

	err := flush(batch, int64(to))
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
	eventKeys := &dymint.EventKeys{}
	err = eventKeys.Unmarshal(keysList)
	if err != nil {
		return pruned, err
	}
	for _, key := range eventKeys.Keys {
		pruned++
		err := batch.Delete(key)
		if err != nil {
			continue
		}

	}
	return pruned, nil
}

func (idx *BlockerIndexer) addEventKeys(height int64, beginKeys *dymint.EventKeys, endKeys *dymint.EventKeys, batch store.KVBatch) error {
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
