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

type TxIndex struct {
	store store.KV
}

func NewTxIndex(store store.KV) *TxIndex {
	return &TxIndex{
		store: store,
	}
}

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

func (txi *TxIndex) AddBatch(b *txindex.Batch) error {
	storeBatch := txi.store.NewBatch()
	defer storeBatch.Discard()

	var eventKeysBatch dmtypes.EventKeys
	for _, result := range b.Ops {
		hash := types.Tx(result.Tx).Hash()

		eventKeys, err := txi.indexEvents(result, hash, storeBatch)
		if err != nil {
			return err
		}
		eventKeysBatch.Keys = append(eventKeysBatch.Keys, eventKeys.Keys...)

		err = storeBatch.Set(keyForHeight(result), hash)
		if err != nil {
			return err
		}

		rawBytes, err := proto.Marshal(result)
		if err != nil {
			return err
		}

		err = storeBatch.Set(hash, rawBytes)
		if err != nil {
			return err
		}
	}

	err := txi.addEventKeys(b.Height, &eventKeysBatch, storeBatch)
	if err != nil {
		return err
	}

	return storeBatch.Commit()
}

func (txi *TxIndex) Index(result *abci.TxResult) error {
	b := txi.store.NewBatch()
	defer b.Discard()

	hash := types.Tx(result.Tx).Hash()

	eventKeys, err := txi.indexEvents(result, hash, b)
	if err != nil {
		return err
	}

	err = txi.addEventKeys(result.Height, &eventKeys, b)
	if err != nil {
		return nil
	}

	err = b.Set(keyForHeight(result), hash)
	if err != nil {
		return err
	}

	rawBytes, err := proto.Marshal(result)
	if err != nil {
		return err
	}

	err = b.Set(hash, rawBytes)
	if err != nil {
		return err
	}

	return b.Commit()
}

func (txi *TxIndex) indexEvents(result *abci.TxResult, hash []byte, store store.KVBatch) (dmtypes.EventKeys, error) {
	eventKeys := dmtypes.EventKeys{}
	for _, event := range result.Result.Events {

		if len(event.Type) == 0 {
			continue
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

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

func (txi *TxIndex) Search(ctx context.Context, q *query.Query) ([]*abci.TxResult, error) {
	select {
	case <-ctx.Done():
		return make([]*abci.TxResult, 0), nil

	default:
	}

	var hashesInitialized bool
	filteredHashes := make(map[string][]byte)

	conditions, err := q.Conditions()
	if err != nil {
		return nil, fmt.Errorf("during parsing conditions from query: %w", err)
	}

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

	skipIndexes := make([]int, 0)

	ranges, rangeIndexes := indexer.LookForRanges(conditions)
	if len(ranges) > 0 {
		skipIndexes = append(skipIndexes, rangeIndexes...)

		for _, qr := range ranges {
			if !hashesInitialized {
				filteredHashes = txi.matchRange(ctx, qr, startKey(qr.Key), filteredHashes, true)
				hashesInitialized = true

				if len(filteredHashes) == 0 {
					break
				}
			} else {
				filteredHashes = txi.matchRange(ctx, qr, startKey(qr.Key), filteredHashes, false)
			}
		}
	}

	height := lookForHeight(conditions)

	for i, c := range conditions {
		if intInSlice(i, skipIndexes) {
			continue
		}

		if !hashesInitialized {
			filteredHashes = txi.match(ctx, c, startKeyForCondition(c, height), filteredHashes, true)
			hashesInitialized = true

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

func lookForHeight(conditions []query.Condition) (height int64) {
	for _, c := range conditions {
		if c.CompositeKey == tmtypes.TxHeightKey && c.Op == query.OpEqual {
			return c.Operand.(int64)
		}
	}
	return 0
}

func (txi *TxIndex) match(
	ctx context.Context,
	c query.Condition,
	startKeyBz []byte,
	filteredHashes map[string][]byte,
	firstRun bool,
) map[string][]byte {
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

		it := txi.store.PrefixIterator(startKey(c.CompositeKey))
		defer it.Discard()

		for ; it.Valid(); it.Next() {
			cont := true

			tmpHashes[string(it.Value())] = it.Value()

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
		return tmpHashes
	}

	for k := range filteredHashes {
		cont := true

		if tmpHashes[k] == nil {
			delete(filteredHashes, k)

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

func (txi *TxIndex) matchRange(
	ctx context.Context,
	qr indexer.QueryRange,
	startKey []byte,
	filteredHashes map[string][]byte,
	firstRun bool,
) map[string][]byte {
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
		panic(err)
	}

	if len(tmpHashes) == 0 || firstRun {
		return tmpHashes
	}

	for k := range filteredHashes {
		cont := true

		if tmpHashes[k] == nil {
			delete(filteredHashes, k)

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
	pruned, err := txi.pruneTxsAndEvents(from, to, logger)
	if err != nil {
		return 0, err
	}
	return pruned, nil
}

func (txi *TxIndex) pruneTxsAndEvents(from, to uint64, logger log.Logger) (uint64, error) {
	pruned := uint64(0)
	toFlush := uint64(0)
	batch := txi.store.NewBatch()
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
			batch = txi.store.NewBatch()
			toFlush = 0
		}

		prunedEvents, err := txi.pruneEvents(h, batch)
		pruned += prunedEvents
		toFlush += prunedEvents
		if err != nil {
			continue
		}

		it := txi.store.PrefixIterator(prefixForHeight(h))

		for ; it.Valid(); it.Next() {
			toFlush++
			if err := batch.Delete(it.Key()); err != nil {
				continue
			}
			if err := batch.Delete(it.Value()); err != nil {
				continue
			}
			pruned++
		}

		it.Discard()

	}

	err := flush(batch, int64(to))
	if err != nil {
		return 0, err
	}

	return pruned, nil
}

func (txi *TxIndex) pruneEvents(height int64, batch store.KVBatch) (uint64, error) {
	pruned := uint64(0)
	eventKey, err := eventHeightKey(height)
	if err != nil {
		return pruned, err
	}
	keysList, err := txi.store.Get(eventKey)
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
			return pruned, err
		}
	}
	return pruned, nil
}

func (txi *TxIndex) addEventKeys(height int64, eventKeys *dymint.EventKeys, batch store.KVBatch) error {
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
	return []byte(fmt.Sprintf("%s/%d/%d",
		tmtypes.TxHeightKey,
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
