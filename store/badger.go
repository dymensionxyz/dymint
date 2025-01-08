package store

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/dymensionxyz/dymint/types"
	"github.com/dymensionxyz/gerr-cosmos/gerrc"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

const (
	gcTimeout    = 1 * time.Minute
	discardRatio = 0.5 // Recommended by badger. Indicates that a file will be rewritten if half the space can be discarded.
)

var (
	_ KV      = &BadgerKV{}
	_ KVBatch = &BadgerBatch{}
)

// BadgerKV is a implementation of KVStore using Badger v3.
type BadgerKV struct {
	db        *badger.DB
	closing   chan struct{}
	closeOnce sync.Once
}

// NewDefaultInMemoryKVStore builds KVStore that works in-memory (without accessing disk).
func NewDefaultInMemoryKVStore() KV {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		panic(err)
	}
	return &BadgerKV{
		db:      db,
		closing: make(chan struct{}),
	}
}

func NewKVStore(rootDir, dbPath, dbName string,
	badgerOpts BadgerOpts,

	logger types.Logger) KV {
	path := filepath.Join(Rootify(rootDir, dbPath), dbName)
	opts := memoryEfficientBadgerConfig(path, badgerOpts)
	db, err := badger.Open(*opts)
	if err != nil {
		panic(err)
	}
	b := &BadgerKV{
		db:      db,
		closing: make(chan struct{}),
	}
	go b.gc(gcTimeout, discardRatio, logger)
	return b
}

// NewDefaultKVStore creates instance of default key-value store.
func NewDefaultKVStore(rootDir, dbPath, dbName string) KV {
	return NewKVStore(rootDir, dbPath, dbName, BadgerOpts{SyncWrites: true}, log.NewNopLogger())
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
	b.closeOnce.Do(func() {
		close(b.closing)
	})
	return b.db.Close()
}

func (b *BadgerKV) gc(period time.Duration, discardRatio float64, logger types.Logger) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-b.closing:
			// Exit the periodic garbage collector function when store is closed
			return
		case <-ticker.C:
			err := b.db.RunValueLogGC(discardRatio)
			if err != nil {
				logger.Debug("Running db RunValueLogGC", "err", err)
				continue
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

type BadgerOpts struct {
	SyncWrites    bool
	NumCompactors int
}

// memoryEfficientBadgerConfig sets badger configuration parameters to reduce memory usage, specially during compactions to avoid memory spikes that causes OOM.
// based on https://github.com/celestiaorg/celestia-node/issues/2905
func memoryEfficientBadgerConfig(path string, o BadgerOpts) *badger.Options {
	opts := badger.DefaultOptions(path) // this must be copied
	// SyncWrites is a configuration option in Badger that determines whether writes are immediately synced to disk or no.
	// If set to true it writes to the write-ahead log (value log) are synced to disk before being applied to the LSM tree.
	opts.SyncWrites = o.SyncWrites
	// default 64mib => 0 - disable block cache
	// BlockCacheSize specifies how much data cache should hold in memory.
	// It improves lookup performance but increases memory consumption.
	// Not really necessary if disabling compression
	opts.BlockCacheSize = 0
	// compressions reduces storage usage but increases memory consumption, specially during compaction
	opts.Compression = options.None
	// MemTables: maximum size of in-memory data structures  before they are flushed to disk
	// default 64mib => 16mib - decreases memory usage and makes compaction more often
	opts.MemTableSize = 16 << 20
	// NumMemtables is a configuration option in Badger that sets the maximum number of memtables to keep in memory before stalling
	// default 5 => 3
	opts.NumMemtables = 3
	// NumLevelZeroTables sets the maximum number of Level 0 tables before compaction starts
	// default 5 => 3
	opts.NumLevelZeroTables = 3
	// default 15 => 5 - this prevents memory growth on CPU constraint systems by blocking all writers
	opts.NumLevelZeroTablesStall = 5
	// reducing number compactors, makes it slower but reduces memory usage during compaction
	opts.NumCompactors = o.NumCompactors
	if opts.NumCompactors == 0 {
		opts.NumCompactors = 2 // default
	}
	// makes sure badger is always compacted on shutdown
	opts.CompactL0OnClose = true

	return &opts
}
