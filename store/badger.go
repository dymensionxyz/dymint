package store

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

const (
	gcTimeout    = 1 * time.Minute
	discardRatio = 0.125
)

var (
	_ KV      = &BadgerKV{}
	_ KVBatch = &BadgerBatch{}
)

// BadgerKV is a implementation of KVStore using Badger v3.
type BadgerKV struct {
	db *badger.DB
}

// NewDefaultInMemoryKVStore builds KVStore that works in-memory (without accessing disk).
func NewDefaultInMemoryKVStore() KV {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}
	return &BadgerKV{
		db: db,
	}
}

func NewKVStore(rootDir, dbPath, dbName string, syncWrites bool) KV {
	path := filepath.Join(Rootify(rootDir, dbPath), dbName)
	opts := constraintBadgerConfig(path)
	db, err := badger.Open(opts.WithSyncWrites(syncWrites))
	if err != nil {
		panic(err)
	}
	b := &BadgerKV{
		db: db,
	}
	go b.gc(gcTimeout, discardRatio)

	return b
}

// NewDefaultKVStore creates instance of default key-value store.
func NewDefaultKVStore(rootDir, dbPath, dbName string) KV {
	return NewKVStore(rootDir, dbPath, dbName, true)
}

// Rootify is helper function to make config creation independent of root dir
func Rootify(rootDir, dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	return filepath.Join(rootDir, dbPath)
}

// Close implements KVStore.
func (b *BadgerKV) Close() error {
	return b.db.Close()
}

func (b *BadgerKV) gc(period time.Duration, discardRatio float64) error {

	gcTimeout := time.NewTimer(period)
	defer gcTimeout.Stop()

	ctx := context.Background()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-gcTimeout.C:
			err := b.db.RunValueLogGC(discardRatio)
			if err != nil {
				return err
			}
		}
	}
}

// Get returns value for given key, or error.
func (b *BadgerKV) Get(key []byte) ([]byte, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()
	item, err := txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, gerrc.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

// Set saves key-value mapping in store.
func (b *BadgerKV) Set(key []byte, value []byte) error {
	txn := b.db.NewTransaction(true)
	defer txn.Discard()
	err := txn.Set(key, value)
	if err != nil {
		return err
	}
	return txn.Commit()
}

// Delete removes key and corresponding value from store.
func (b *BadgerKV) Delete(key []byte) error {
	txn := b.db.NewTransaction(true)
	defer txn.Discard()
	err := txn.Delete(key)
	if err != nil {
		return err
	}
	return txn.Commit()
}

// NewBatch creates new batch.
// Note: badger batches should be short lived as they use extra resources.
func (b *BadgerKV) NewBatch() KVBatch {
	return &BadgerBatch{
		txn: b.db.NewTransaction(true),
	}
}

// BadgerBatch encapsulates badger transaction
type BadgerBatch struct {
	txn *badger.Txn
}

// Set accumulates key-value entries in a transaction
func (bb *BadgerBatch) Set(key, value []byte) error {
	if err := bb.txn.Set(key, value); err != nil {
		return err
	}

	return nil
}

// Delete removes the key and associated value from store
func (bb *BadgerBatch) Delete(key []byte) error {
	return bb.txn.Delete(key)
}

// Commit commits a transaction
func (bb *BadgerBatch) Commit() error {
	return bb.txn.Commit()
}

// Discard cancels a transaction
func (bb *BadgerBatch) Discard() {
	bb.txn.Discard()
}

var _ KVIterator = &BadgerIterator{}

// PrefixIterator returns instance of prefix Iterator for BadgerKV.
func (b *BadgerKV) PrefixIterator(prefix []byte) KVIterator {
	txn := b.db.NewTransaction(false)
	iter := txn.NewIterator(badger.DefaultIteratorOptions)
	iter.Seek(prefix)
	return &BadgerIterator{
		txn:       txn,
		iter:      iter,
		prefix:    prefix,
		lastError: nil,
	}
}

// BadgerIterator encapsulates prefix iterator for badger kv store.
type BadgerIterator struct {
	txn       *badger.Txn
	iter      *badger.Iterator
	prefix    []byte
	lastError error
}

// Valid returns true if iterator is inside its prefix, false otherwise.
func (i *BadgerIterator) Valid() bool {
	return i.iter.ValidForPrefix(i.prefix)
}

// Next progresses iterator to the next key-value pair.
func (i *BadgerIterator) Next() {
	i.iter.Next()
}

// Key returns key pointed by iterator.
func (i *BadgerIterator) Key() []byte {
	return i.iter.Item().KeyCopy(nil)
}

// Value returns value pointer by iterator.
func (i *BadgerIterator) Value() []byte {
	val, err := i.iter.Item().ValueCopy(nil)
	if err != nil {
		i.lastError = err
	}
	return val
}

// Error returns last error that occurred during iteration.
func (i *BadgerIterator) Error() error {
	return i.lastError
}

// Discard has to be called to free iterator resources.
func (i *BadgerIterator) Discard() {
	i.iter.Close()
	i.txn.Discard()
}

func constraintBadgerConfig(path string) *badger.Options {
	opts := badger.DefaultOptions(path) // this must be copied
	// ValueLog:
	// 2mib default => share.Size - makes sure headers and samples are stored in value log
	// This *tremendously* reduces the amount of memory used by the node, up to 10 times less during
	// compaction
	opts.ValueThreshold = share.Size
	// make sure we don't have any limits for stored headers
	opts.ValueLogMaxEntries = 100000000

	// badger stores checksum for every value, but doesn't verify it by default
	// enabling this option may allow us to see detect corrupted data
	opts.ChecksumVerificationMode = options.OnBlockRead
	opts.VerifyValueChecksum = true
	// default 64mib => 0 - disable block cache
	// most of our component maintain their own caches, so this is not needed
	opts.BlockCacheSize = 0
	// not much gain as it compresses the LSM only as well compression requires block cache
	opts.Compression = options.None

	// MemTables:
	// default 64mib => 16mib - decreases memory usage and makes compaction more often
	opts.MemTableSize = 16 << 20
	// default 5 => 3
	opts.NumMemtables = 3
	// default 5 => 3
	opts.NumLevelZeroTables = 3
	// default 15 => 5 - this prevents memory growth on CPU constraint systems by blocking all writers
	opts.NumLevelZeroTablesStall = 5

	opts.NumCompactors = 2
	// makes sure badger is always compacted on shutdown
	opts.CompactL0OnClose = true

	return &opts
}
