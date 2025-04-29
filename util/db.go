package util

import (
	"github.com/syndtr/goleveldb/leveldb"
)

type DBManager struct {
    db *leveldb.DB
}

var (
	db *DBManager
)

// TODO: move const define to delicate file
const (
    AccountBalancePrefix       byte = 0x01 // Prefix for user-related data
    HashBlockFilePrefix byte = 0x02 // Prefix for transaction-related data
    HashHeightPrefix       byte = 0x03 // Prefix for block-related data
	HashHeadPerfix byte = 0x04
	HashUndo byte = 0x05
	HashTip byte = 0x06
	HeightHash = 0x07

	BlockDir = "./blocks/"
)

// NewDB initializes and returns a new DBManager instance
func NewDB(path string) (*DBManager, error) {
    db, err := leveldb.OpenFile(path, nil) // Open the database
    if err != nil {
    	return nil, err
    }
    return &DBManager{db: db}, nil
}

// Close the database instance
func (manager *DBManager) Close() error {
    if manager.db != nil {
        return manager.db.Close()
    }
    return nil
}

// Insert adds a key-value pair to the database
func (manager *DBManager) Insert(key, value []byte) error {
    return manager.db.Put(key, value, nil)
}

func (manager *DBManager) BatchInsert(batch *leveldb.Batch) error {
    return manager.db.Write(batch, nil)
}

// Get retrieves a value by key from the database
func (manager *DBManager) Get(key []byte) ([]byte, error) {
    return manager.db.Get(key, nil)
}