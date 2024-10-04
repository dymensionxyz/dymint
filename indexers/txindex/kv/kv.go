package kv

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/tendermint/tendermint/libs/log"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/pubsub/query"
	tmtypes "github.com/tendermint/tendermint/types"

	indexer "github.com/dymensionxyz/dymint/indexers/blockindexer"
	"github.com/dymensionxyz/dymint/indexers/txindex"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/dymint/types/pb/dymint"
	dmtypes "github.com/dymensionxyz/dymint/types/pb/dymint"
)

const (
	tagKeySeparator = "/"
)

var _ txindex.TxIndexer = (*TxIndex)(nil)

// TxIndex is the simplest possible indexer, backed by key-value storage (levelDB).
type TxIndex struct {
	store store.KV
}

// NewTxIndex creates new KV indexer.
func NewTxIndex(store store.KV) *TxIndex {
	return &TxIndex{
		store: store,
	}
}

// Get gets transaction from the TxIndex storage and returns it or nil if the
// transaction is not found.
func (txi *TxIndex) Get(hash []byte) (*abci.TxResult, error) {
	if len(hash) == 0 {
		return nil, txindex.ErrorEmptyHash
	}

	rawBytes, err := txi.store.Get(hash)
	if err != nil {
		return nil, nil
	}
	if rawBytes == nil {
		return nil, nil
	}

	txResult := new(abci.TxResult)
	err = proto.Unmarshal(rawBytes, txResult)
	if err != nil {
		return nil, fmt.Errorf("reading TxResult: %w", err)
	}

	return txResult, nil
}

// AddBatch indexes a batch of transactions using the given list of events. Each
// key that indexed from the tx's events is a composite of the event type and
// the respective attribute's key delimited by a "." (eg. "account.number").
// Any event with an empty type is not indexed.
func (txi *TxIndex) AddBatch(b *txindex.Batch) error {
	storeBatch := txi.store.NewBatch()
	defer storeBatch.Discard()

	for _, result := range b.Ops {
		hash := types.Tx(result.Tx).Hash()

		// index tx by events
		eventKeys, err := txi.indexEvents(result, hash, storeBatch)
		if err != nil {
			return err
		}

		err = txi.addEventKeys(result.Height, &eventKeys, storeBatch)
		if err != nil {
			return err
		}

		// index by height (always)
		err = storeBatch.Set(keyForHeight(result), hash)
		if err != nil {
			return err
		}

		rawBytes, err := proto.Marshal(result)
		if err != nil {
			return err
		}
		// index by hash (always)
		err = storeBatch.Set(hash, rawBytes)
		if err != nil {
			return err
		}
	}

	return storeBatch.Commit()
}

// Index indexes a single transaction using the given list of events. Each key
// that indexed from the tx's events is a composite of the event type and the
// respective attribute's key delimited by a "." (eg. "account.number").
// Any event with an empty type is not indexed.
func (txi *TxIndex) Index(result *abci.TxResult) error {
	b := txi.store.NewBatch()
	defer b.Discard()

	hash := types.Tx(result.Tx).Hash()

	// index tx by events
	eventKeys, err := txi.indexEvents(result, hash, b)
	if err != nil {
		return err
	}

	// add event keys height index
	err = txi.addEventKeys(result.Height, &eventKeys, b)
	if err != nil {
		return nil
	}

	// index by height (always)
	err = b.Set(keyForHeight(result), hash)
	if err != nil {
		return err
	}

	rawBytes, err := proto.Marshal(result)
	if err != nil {
		return err
	}
	// index by hash (always)
	err = b.Set(hash, rawBytes)
	if err != nil {
		return err
	}

	return b.Commit()
}

func (txi *TxIndex) indexEvents(result *abci.TxResult, hash []byte, store store.KVBatch) (dmtypes.EventKeys, error) {
	eventKeys := dmtypes.EventKeys{}
	for _, event := range result.Result.Events {
		// only index events with a non-empty type
		if len(event.Type) == 0 {
			continue
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

			// index if `index: true` is set
			compositeTag := fmt.Sprintf("%s.%s", event.Type, string(attr.Key))
			if attr.GetIndex() {
				err := store.Set(keyForEvent(compositeTag, attr.Value, result), hash)
				if err != nil {
					return dmtypes.EventKeys{}, err
				}
				eventKeys.Keys = append(eventKeys.Keys, keyForEvent(compositeTag, attr.Value, result))
			}
		}
	}

	return eventKeys, nil
}

// Search performs a search using the given query.
//
// It breaks the query into conditions (like "tx.height > 5"). For each
// condition, it queries the DB index. One special use cases here: (1) if
// "tx.hash" is found, it returns tx result for it (2) for range queries it is
// better for the client to provide both lower and upper bounds, so we are not
// performing a full scan. Results from querying indexes are then intersected
// and returned to the caller, in no particular order.
//
// Search will exit early and return any result fetched so far,
// when a message is received on the context chan.
func (txi *TxIndex) Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error) {
	select {
	case <-ctx.Done():
		return make([]*abci.TxResult, 0), nil

	default:
	}

	var hashesInitialized bool
	filteredHashes := make(map[string][]byte)

	// get a list of conditions (like "tx.height > 5")
	conditions, err := q.Conditions()
	if err != nil {
		return nil, fmt.Errorf("during parsing conditions from query: %w", err)
	}

	// if there is a hash condition, return the result immediately
	hash, ok, err := lookForHash(conditions)
	if err != nil {
		return nil, fmt.Errorf("during searching for a hash in the query: %w", err)
	} else if ok {
		res, err := txi.Get(hash)
		switch {
		case err != nil:
			return []*abci.TxResult{}, fmt.Errorf("while retrieving the result: %w", err)
		case res == nil:
			return []*abci.TxResult{}, nil
		default:
			return []*abci.TxResult{res}, nil
		}
	}

	// conditions to skip because they're handled before "everything else"
	skipIndexes := make([]int, 0)

	// extract ranges
	// if both upper and lower bounds exist, it's better to get them in order not
	// no iterate over kvs that are not within range.
	ranges, rangeIndexes := indexer.LookForRanges(conditions)
	if len(ranges) > 0 {
		skipIndexes = append(skipIndexes, rangeIndexes...)

		for _, qr := range ranges {
			if !hashesInitialized {
				filteredHashes = txi.matchRange(ctx, qr, startKey(qr.Key), filteredHashes, true)
				hashesInitialized = true

				// Ignore any remaining conditions if the first condition resulted
				// in no matches (assuming implicit AND operand).
				if len(filteredHashes) == 0 {
					break
				}
			} else {
				filteredHashes = txi.matchRange(ctx, qr, startKey(qr.Key), filteredHashes, false)
			}
		}
	}

	// if there is a height condition ("tx.height=3"), extract it
	height := lookForHeight(conditions)

	// for all other conditions
	for i, c := range conditions {
		if intInSlice(i, skipIndexes) {
			continue
		}

		if !hashesInitialized {
			filteredHashes = txi.match(ctx, c, startKeyForCondition(c, height), filteredHashes, true)
			hashesInitialized = true

			// Ignore any remaining conditions if the first condition resulted
			// in no matches (assuming implicit AND operand).
			if len(filteredHashes) == 0 {
				break
			}
		} else {
			filteredHashes = txi.match(ctx, c, startKeyForCondition(c, height), filteredHashes, false)
		}
	}

	results := make([]*abci.TxResult, 0, len(filteredHashes))
	for _, h := range filteredHashes {
		cont := true

		res, err := txi.Get(h)
		if err != nil {
			return nil, fmt.Errorf("get Tx{%X}: %w", h, err)
		}
		if res == nil {
			continue
		}
		results = append(results, res)

		// Potentially exit early.
		select {
		case <-ctx.Done():
			cont = false
		default:
		}

		if !cont {
			break
		}
	}

	return results, nil
}

func lookForHash(conditions []query.Condition) (hash []byte, ok bool, err error) {
	for _, c := range conditions {
		if c.CompositeKey == tmtypes.TxHashKey {
			decoded, err := hex.DecodeString(c.Operand.(string))
			return decoded, true, err
		}
	}
	return
}

// lookForHeight returns a height if there is an "height=X" condition.
func lookForHeight(conditions []query.Condition) (height int64) {
	for _, c := range conditions {
		if c.CompositeKey == tmtypes.TxHeightKey && c.Op == query.OpEqual {
			return c.Operand.(int64)
		}
	}
	return 0
}

// match returns all matching txs by hash that meet a given condition and start
// key. An already filtered result (filteredHashes) is provided such that any
// non-intersecting matches are removed.
//
// NOTE: filteredHashes may be empty if no previous condition has matched.
func (txi *TxIndex) match(
	ctx context.Context,
	c query.Condition,
	startKeyBz []byte,
	filteredHashes map[string][]byte,
	firstRun bool,
) map[string][]byte {
	// A previous match was attempted but resulted in no matches, so we return
	// no matches (assuming AND operand).
	if !firstRun && len(filteredHashes) == 0 {
		return filteredHashes
	}

	tmpHashes := make(map[string][]byte)

	switch {
	case c.Op == query.OpEqual:
		it := txi.store.PrefixIterator(startKeyBz)
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			tmpHashes[string(it.Value())] = it.Value()

			// Potentially exit early.
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
			panic(err)
		}

	case c.Op == query.OpExists:
		// XXX: can't use startKeyBz here because c.Operand is nil
		// (e.g. "account.owner/<nil>/" won't match w/ a single row)
		it := txi.store.PrefixIterator(startKey(c.CompositeKey))
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			tmpHashes[string(it.Value())] = it.Value()

			// Potentially exit early.
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
			panic(err)
		}

	case c.Op == query.OpContains:
		// XXX: startKey does not apply here.
		// For example, if startKey = "account.owner/an/" and search query = "account.owner CONTAINS an"
		// we can't iterate with prefix "account.owner/an/" because we might miss keys like "account.owner/Ulan/"
		it := txi.store.PrefixIterator(startKey(c.CompositeKey))
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			if !isTagKey(it.Key()) {
				continue
			}

			if strings.Contains(extractValueFromKey(it.Key()), c.Operand.(string)) {
				tmpHashes[string(it.Value())] = it.Value()
			}

			// Potentially exit early.
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
			panic(err)
		}
	default:
		panic("other operators should be handled already")
	}

	if len(tmpHashes) == 0 || firstRun {
		// Either:
		//
		// 1. Regardless if a previous match was attempted, which may have had
		// results, but no match was found for the current condition, then we
		// return no matches (assuming AND operand).
		//
		// 2. A previous match was not attempted, so we return all results.
		return tmpHashes
	}

	// Remove/reduce matches in filteredHashes that were not found in this
	// match (tmpHashes).
	for k := range filteredHashes {
		cont := true

		if tmpHashes[k] == nil {
			delete(filteredHashes, k)

			// Potentially exit early.
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

	return filteredHashes
}

// matchRange returns all matching txs by hash that meet a given queryRange and
// start key. An already filtered result (filteredHashes) is provided such that
// any non-intersecting matches are removed.
//
// NOTE: filteredHashes may be empty if no previous condition has matched.
func (txi *TxIndex) matchRange(
	ctx context.Context,
	qr indexer.QueryRange,
	startKey []byte,
	filteredHashes map[string][]byte,
	firstRun bool,
) map[string][]byte {
	// A previous match was attempted but resulted in no matches, so we return
	// no matches (assuming AND operand).
	if !firstRun && len(filteredHashes) == 0 {
		return filteredHashes
	}

	tmpHashes := make(map[string][]byte)
	lowerBound := qr.LowerBoundValue()
	upperBound := qr.UpperBoundValue()

	it := txi.store.PrefixIterator(startKey)
	defer it.Discard()

LOOP:
	for ; it.Valid(); it.Next() {
		cont := true

		if !isTagKey(it.Key()) {
			continue
		}

		if _, ok := qr.AnyBound().(int64); ok {
			v, err := strconv.ParseInt(extractValueFromKey(it.Key()), 10, 64)
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
				tmpHashes[string(it.Value())] = it.Value()
			}

			// XXX: passing time in a ABCI Events is not yet implemented
			// case time.Time:
			// 	v := strconv.ParseInt(extractValueFromKey(it.Key()), 10, 64)
			// 	if v == r.upperBound {
			// 		break
			// 	}
		}

		// Potentially exit early.
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
		panic(err)
	}

	if len(tmpHashes) == 0 || firstRun {
		// Either:
		//
		// 1. Regardless if a previous match was attempted, which may have had
		// results, but no match was found for the current condition, then we
		// return no matches (assuming AND operand).
		//
		// 2. A previous match was not attempted, so we return all results.
		return tmpHashes
	}

	// Remove/reduce matches in filteredHashes that were not found in this
	// match (tmpHashes).
	for k := range filteredHashes {
		cont := true

		if tmpHashes[k] == nil {
			delete(filteredHashes, k)

			// Potentially exit early.
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

	return filteredHashes
}

func (txi *TxIndex) Prune(from, to uint64, logger log.Logger) (uint64, error) {
	pruned, err := txi.pruneTxs(from, to)
	if err != nil {
		return 0, err
	}
	return pruned, nil
}

func (txi *TxIndex) pruneTxs(from, to uint64) (uint64, error) {
	pruned := uint64(0)
	batch := txi.store.NewBatch()
	defer batch.Discard()

	flush := func(batch store.KVBatch, height int64) error {
		err := batch.Commit()
		if err != nil {
			return fmt.Errorf("flush batch to disk: height %d: %w", height, err)
		}
		return nil
	}

	for h := from; h < to; h++ {

		it := txi.store.PrefixIterator(prefixForHeight(int64(h)))
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			if err := batch.Delete(it.Key()); err != nil {
				continue
			}
			if err := batch.Delete(it.Value()); err != nil {
				continue
			}
			if err := txi.pruneEvents(h, batch); err != nil {
				continue
			}
			pruned++
			// flush every 1000 txs to avoid batches becoming too large
			if pruned%1000 == 0 && pruned > 0 {
				err := flush(batch, int64(h))
				if err != nil {
					return 0, err
				}
				batch.Discard()
				batch = txi.store.NewBatch()
			}
		}
	}

	err := flush(batch, int64(to))
	if err != nil {
		return 0, err
	}

	return pruned, nil
}

func (txi *TxIndex) pruneEvents(height uint64, batch store.KVBatch) error {
	eventKey, err := eventHeightKey(int64(height))
	if err != nil {
		return err
	}
	keysList, err := txi.store.Get(eventKey)
	if err != nil {
		return err
	}
	eventKeys := &dymint.EventKeys{}
	err = eventKeys.Unmarshal(keysList)
	if err != nil {
		return err
	}
	for _, key := range eventKeys.Keys {
		err := batch.Delete(key)
		if err != nil {
			return err
		}
	}
	return nil
}

func (txi *TxIndex) addEventKeys(height int64, eventKeys *dymint.EventKeys, batch store.KVBatch) error {
	// index event keys by height
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

// Keys

func isTagKey(key []byte) bool {
	return strings.Count(string(key), tagKeySeparator) == 3
}

func extractValueFromKey(key []byte) string {
	parts := strings.SplitN(string(key), tagKeySeparator, 3)
	return parts[1]
}

func keyForEvent(key string, value []byte, result *abci.TxResult) []byte {
	return []byte(fmt.Sprintf("%s/%s/%d/%d",
		key,
		value,
		result.Height,
		result.Index,
	))
}

func keyForHeight(result *abci.TxResult) []byte {
	return []byte(fmt.Sprintf("%s/%d/%d/%d",
		tmtypes.TxHeightKey,
		result.Height,
		result.Height,
		result.Index,
	))
}

func startKeyForCondition(c query.Condition, height int64) []byte {
	if height > 0 {
		return startKey(c.CompositeKey, c.Operand, height)
	}
	return startKey(c.CompositeKey, c.Operand)
}

func startKey(fields ...interface{}) []byte {
	var b bytes.Buffer
	for _, f := range fields {
		b.Write([]byte(fmt.Sprintf("%v", f) + tagKeySeparator))
	}
	return b.Bytes()
}

func prefixForHeight(height int64) []byte {
	return []byte(fmt.Sprintf("%s/%d/",
		tmtypes.TxHeightKey,
		height,
	))
}
