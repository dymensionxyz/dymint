package store

import (
	"path/filepath"

	"github.com/dgraph-io/badger/v3"
)

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

// NewDefaultKVStore creates instance of default key-value store.
func NewDefaultKVStore(rootDir, dbPath, dbName string) KV {
	path := filepath.Join(rootify(rootDir, dbPath), dbName)
	db, err := badger.Open(badger.DefaultOptions(path).WithSyncWrites(true))
	if err != nil {
		panic(err)
	}
	return &BadgerKV{
		db: db,
	}
}

// rootify works just like in cosmos-sdk
func rootify(rootDir, dbPath string) string {
	if filepath.IsAbs(dbPath) {
		return dbPath
	}
	return filepath.Join(rootDir, dbPath)
}
