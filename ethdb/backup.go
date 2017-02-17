package ethdb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// This file contains the methods and interfaces used to support the backup
// mechanism on the eth state server without having to deal with vendoring.
// Many of the leveldb database object's methods require custom option types
// which result in ugly vendored types.

type LDBIter interface {
	iterator.Iterator
}

type LDBSnapshot struct {
	*leveldb.Snapshot
}

func (db *LDBDatabase) LDBSnapshot() (*LDBSnapshot, error) {
	snap, err := db.LDB().GetSnapshot()
	return &LDBSnapshot{snap}, err
}

func (snap *LDBSnapshot) FullIter() LDBIter {
	return snap.NewIterator(&util.Range{}, &opt.ReadOptions{DontFillCache: true})
}

type RawLDB struct {
	*leveldb.DB
}

// Open a LDB, errors if new.
// Using similar settings to eth's own internal ldb
func OpenNewRawLDB(fileName string) (*RawLDB, error) {
	db, err := leveldb.OpenFile(fileName, &opt.Options{
		ErrorIfExist:       true,
		BlockCacheCapacity: 32 * opt.MiB,
		WriteBuffer:        32 * opt.MiB})
	return &RawLDB{db}, err
}

// Clone of underlying Put, but without the write options. Sync is set by a
// bool
func (rdb *RawLDB) Put(key, value []byte, sync bool) error {
	return rdb.DB.Put(key, value, &opt.WriteOptions{Sync: sync})
}

func (rdb *RawLDB) WriteBatch(batch *RawLDBBatch, sync bool) error {
	if batch != nil && batch.Batch != nil {
		return rdb.DB.Write(batch.Batch, &opt.WriteOptions{Sync: sync})
	}
	return nil
}

func (rdb *RawLDB) CompactAll() error {
	return rdb.DB.CompactRange(util.Range{})
}

type RawLDBBatch struct {
	*leveldb.Batch
}

func NewRawLDBBatch() *RawLDBBatch {
	return &RawLDBBatch{new(leveldb.Batch)}
}
