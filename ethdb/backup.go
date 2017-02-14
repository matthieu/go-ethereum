package ethdb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// This file contains the methods and interfaces used to support the backup
// mechanism on the eth state server without having to deal with vendoring

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
func OpenNewRawLDB(fileName string) (*RawLDB, error) {
	db, err := leveldb.OpenFile(fileName, &opt.Options{ErrorIfExist: true})
	return &RawLDB{db}, err
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
