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
	discardRatio = 0.5
)

var (
	_ KV      = &BadgerKV{}
	_ KVBatch = &BadgerBatch{}
)

type BadgerKV struct {
	db        *badger.DB
	closing   chan struct{}
	closeOnce sync.Once
}

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

func NewKVStore(rootDir, dbPath, dbName string, syncWrites bool, logger types.Logger) KV {
	path := filepath.Join(Rootify(rootDir, dbPath), dbName)
	opts := memoryEfficientBadgerConfig(path, syncWrites)
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

func NewDefaultKVStore(rootDir, dbPath, dbName string) KV {
	return NewKVStore(rootDir, dbPath, dbName, true, log.NewNopLogger())
}

func Rootify(rootDir, dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	return filepath.Join(rootDir, dbPath)
}

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

			return
		case <-ticker.C:
			err := b.db.RunValueLogGC(discardRatio)
			if err != nil {
				continue
			}
		}
	}
}

func (b *BadgerKV) Get(key []byte) ([]byte, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard() // commit is more correct
	item, err := txn.Get(key)
	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, gerrc.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return item.ValueCopy(nil)
}

func (b *BadgerKV) Set(key []byte, value []byte) error {
	txn := b.db.NewTransaction(true)
	defer txn.Discard()
	err := txn.Set(key, value)
	if err != nil {
		return err
	}
	return txn.Commit()
}

func (b *BadgerKV) Delete(key []byte) error {
	txn := b.db.NewTransaction(true)
	defer txn.Discard()
	err := txn.Delete(key)
	if err != nil {
		return err
	}
	return txn.Commit()
}

func (b *BadgerKV) NewBatch() KVBatch {
	return &BadgerBatch{
		txn: b.db.NewTransaction(true),
	}
}

type BadgerBatch struct {
	txn *badger.Txn
}

func (bb *BadgerBatch) Set(key, value []byte) error {
	if err := bb.txn.Set(key, value); err != nil {
		return err
	}

	return nil
}

func (bb *BadgerBatch) Delete(key []byte) error {
	return bb.txn.Delete(key)
}

func (bb *BadgerBatch) Commit() error {
	return bb.txn.Commit()
}

func (bb *BadgerBatch) Discard() {
	bb.txn.Discard()
}

var _ KVIterator = &BadgerIterator{}

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

type BadgerIterator struct {
	txn       *badger.Txn
	iter      *badger.Iterator
	prefix    []byte
	lastError error
}

func (i *BadgerIterator) Valid() bool {
	return i.iter.ValidForPrefix(i.prefix)
}

func (i *BadgerIterator) Next() {
	i.iter.Next()
}

func (i *BadgerIterator) Key() []byte {
	return i.iter.Item().KeyCopy(nil)
}

func (i *BadgerIterator) Value() []byte {
	val, err := i.iter.Item().ValueCopy(nil)
	if err != nil {
		i.lastError = err
	}
	return val
}

func (i *BadgerIterator) Error() error {
	return i.lastError
}

func (i *BadgerIterator) Discard() {
	i.iter.Close()
	i.txn.Discard()
}

func memoryEfficientBadgerConfig(path string, syncWrites bool) *badger.Options {
	opts := badger.DefaultOptions(path)

	opts.SyncWrites = syncWrites

	opts.BlockCacheSize = 0

	opts.Compression = options.None

	opts.MemTableSize = 16 << 20

	opts.NumMemtables = 3

	opts.NumLevelZeroTables = 3

	opts.NumLevelZeroTablesStall = 5

	opts.NumCompactors = 2

	opts.CompactL0OnClose = true

	return &opts
}
