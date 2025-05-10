package db

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"

	"github.com/nanlour/da/src/block"
	"github.com/syndtr/goleveldb/leveldb"
)

type DBManager struct {
	db *leveldb.DB
}

// TODO: move const define to delicate file
const (
	accountBalancePrefix byte = 0x01 // Prefix for user-related data
	hashBlockPerfix      byte = 0x02
	tipHash              byte = 0x03
)

func PrefixKey(prefix byte, data []byte) []byte {
	result := make([]byte, 1+len(data))
	result[0] = prefix
	copy(result[1:], data)
	return result
}

// InitialDB initializes and returns a new DBManager instance
func InitialDB(path string) (*DBManager, error) {
	db, err := leveldb.OpenFile(path, nil) // Open the database
	if err != nil {
		log.Fatalf("Failed to open db: %v", err)
		return nil, err
	}
	mainDB := &DBManager{db: db}
	return mainDB, nil
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

// Get retrieves a value by key from the database
func (manager *DBManager) Get(key []byte) ([]byte, error) {
	return manager.db.Get(key, nil)
}

// Account Balance functions (float64)
func (manager *DBManager) GetAccountBalance(address *[32]byte) (float64, error) {
	key := PrefixKey(accountBalancePrefix, address[:])
	data, err := manager.Get(key)
	if err != nil {
		return 0, err
	}

	bits := binary.LittleEndian.Uint64(data)
	return math.Float64frombits(bits), nil
}

func (manager *DBManager) InsertAccountBalance(address *[32]byte, balance float64) error {
	key := PrefixKey(accountBalancePrefix, address[:])

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(balance))

	return manager.Insert(key, buf)
}

// GetHashBlockretrieves a Block for a given block hash
func (manager *DBManager) GetHashBlock(hash []byte) (*block.Block, error) {
	// Create prefixed key
	key := PrefixKey(hashBlockPerfix, hash[:])

	// Get serialized data from the database
	data, err := manager.Get(key)
	if err != nil {
		return nil, err
	}

	// Deserialize the data into a BlockHead object
	blockHead := &block.Block{}
	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, blockHead)
	if err != nil {
		return nil, err
	}

	return blockHead, nil
}

// InsertHashBlock stores a Block for a given block hash
func (manager *DBManager) InsertHashBlock(hash *[32]byte, block *block.Block) error {
	// Create prefixed key
	key := PrefixKey(hashBlockPerfix, hash[:])

	// Serialize the BlockHead object
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, block)
	if err != nil {
		return err
	}

	// Store in database
	return manager.Insert(key, buf.Bytes())
}

// Tip Hash functions
func (manager *DBManager) GetTipHash() ([]byte, error) {
	return manager.Get([]byte{tipHash})
}

func (manager *DBManager) InsertTipHash(hash *[32]byte) error {
	return manager.Insert([]byte{tipHash}, hash[:])
}
