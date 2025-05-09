package db

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/nanlour/da/src/block"
)

// createTempDB creates a temporary database for testing
func createTempDB(t *testing.T) (*DBManager, string) {
	tempDir, err := os.MkdirTemp("", "db_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	dbPath := filepath.Join(tempDir, "testdb")
	manager, err := InitialDB(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return manager, tempDir
}

// TestInitialDBAndClose tests database initialization and closing
func TestInitialDBAndClose(t *testing.T) {
	manager, tempDir := createTempDB(t)
	defer os.RemoveAll(tempDir)

	if err := manager.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}
}

// TestInsertAndGet tests the basic insert and get operations
func TestInsertAndGet(t *testing.T) {
	manager, tempDir := createTempDB(t)
	defer os.RemoveAll(tempDir)
	defer manager.Close()

	// Test data
	key := []byte("testkey")
	value := []byte("testvalue")

	// Test insertion
	if err := manager.Insert(key, value); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	// Test retrieval
	retrieved, err := manager.Get(key)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}

	// Compare retrieved value with original
	if !bytes.Equal(retrieved, value) {
		t.Fatalf("Retrieved value does not match original. Got %v, expected %v", retrieved, value)
	}

	// Test retrieval of non-existent key
	_, err = manager.Get([]byte("nonexistent"))
	if err == nil {
		t.Fatalf("Expected error when retrieving non-existent key")
	}
}

// TestAccountBalance tests account balance operations
func TestAccountBalance(t *testing.T) {
	manager, tempDir := createTempDB(t)
	defer os.RemoveAll(tempDir)
	defer manager.Close()

	// Create test address
	var address [32]byte
	_, err := rand.Read(address[:])
	if err != nil {
		t.Fatalf("Failed to generate random address: %v", err)
	}

	// Test non-existent account balance
	_, err = manager.GetAccountBalance(&address)
	if err == nil {
		t.Fatalf("Expected error when getting balance of non-existent account")
	}

	// Test balance insertion
	balance := 123.45
	err = manager.InsertAccountBalance(&address, balance)
	if err != nil {
		t.Fatalf("Failed to insert account balance: %v", err)
	}

	// Test balance retrieval
	retrieved, err := manager.GetAccountBalance(&address)
	if err != nil {
		t.Fatalf("Failed to retrieve account balance: %v", err)
	}

	// Compare values with small epsilon for floating point comparison
	epsilon := 0.0000001
	if math.Abs(retrieved-balance) > epsilon {
		t.Fatalf("Retrieved balance does not match inserted balance. Got %v, expected %v", retrieved, balance)
	}
}

// TestHashBlock tests block storage and retrieval by hash
func TestHashBlock(t *testing.T) {
	manager, tempDir := createTempDB(t)
	defer os.RemoveAll(tempDir)
	defer manager.Close()

	// Create test block
	testBlock := createTestBlock(t)

	// Generate hash
	blockHash := testBlock.Hash()

	// Test insertion
	err := manager.InsertHashBlock(&blockHash, testBlock)
	if err != nil {
		t.Fatalf("Failed to insert block: %v", err)
	}

	// Test retrieval
	retrievedBlock, err := manager.GetHashBlock(blockHash[:])
	if err != nil {
		t.Fatalf("Failed to retrieve block: %v", err)
	}

	// Compare blocks
	if !compareBlocks(testBlock, retrievedBlock) {
		t.Fatalf("Retrieved block does not match original block")
	}

	// Test retrieval of non-existent block
	var nonExistentHash [32]byte
	_, err = rand.Read(nonExistentHash[:])
	if err != nil {
		t.Fatalf("Failed to generate random hash: %v", err)
	}

	_, err = manager.GetHashBlock(nonExistentHash[:])
	if err == nil {
		t.Fatalf("Expected error when retrieving non-existent block")
	}
}

// TestTipHash tests tip hash storage and retrieval
func TestTipHash(t *testing.T) {
	manager, tempDir := createTempDB(t)
	defer os.RemoveAll(tempDir)
	defer manager.Close()

	// Test retrieval of non-existent tip hash
	_, err := manager.GetTipHash()
	if err == nil {
		t.Fatalf("Expected error when getting non-existent tip hash")
	}

	// Create test hash
	var hash [32]byte
	_, err = rand.Read(hash[:])
	if err != nil {
		t.Fatalf("Failed to generate random hash: %v", err)
	}

	// Test insertion
	err = manager.InsertTipHash(&hash)
	if err != nil {
		t.Fatalf("Failed to insert tip hash: %v", err)
	}

	// Test retrieval
	retrieved, err := manager.GetTipHash()
	if err != nil {
		t.Fatalf("Failed to retrieve tip hash: %v", err)
	}

	// Compare hashes
	if !bytes.Equal(retrieved, hash[:]) {
		t.Fatalf("Retrieved hash does not match original")
	}
}

// Helper function to create a test block
func createTestBlock(t *testing.T) *block.Block {
	// Generate a test private key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create test transaction
	var txn block.Transaction
	_, err = rand.Read(txn.FromAddress[:])
	if err != nil {
		t.Fatalf("Failed to generate random address: %v", err)
	}
	_, err = rand.Read(txn.ToAddress[:])
	if err != nil {
		t.Fatalf("Failed to generate random address: %v", err)
	}
	txn.Amount = 100.5
	txn.Height = 1
	txn.Sign(privKey)

	// Create test block
	var b block.Block
	_, err = rand.Read(b.PreHash[:])
	if err != nil {
		t.Fatalf("Failed to generate random hash: %v", err)
	}
	b.Height = 1
	_, err = rand.Read(b.EpochBeginHash[:])
	if err != nil {
		t.Fatalf("Failed to generate random hash: %v", err)
	}
	b.Txn = txn
	_, err = rand.Read(b.Signature[:])
	if err != nil {
		t.Fatalf("Failed to generate random signature: %v", err)
	}
	_, err = rand.Read(b.PublicKey[:])
	if err != nil {
		t.Fatalf("Failed to generate random public key: %v", err)
	}
	_, err = rand.Read(b.Proof[:])
	if err != nil {
		t.Fatalf("Failed to generate random proof: %v", err)
	}

	return &b
}

// Helper function to compare two blocks
func compareBlocks(a, b *block.Block) bool {
	if !bytes.Equal(a.PreHash[:], b.PreHash[:]) {
		return false
	}
	if a.Height != b.Height {
		return false
	}
	if !bytes.Equal(a.EpochBeginHash[:], b.EpochBeginHash[:]) {
		return false
	}
	if !bytes.Equal(a.Txn.FromAddress[:], b.Txn.FromAddress[:]) {
		return false
	}
	if !bytes.Equal(a.Txn.ToAddress[:], b.Txn.ToAddress[:]) {
		return false
	}
	if a.Txn.Amount != b.Txn.Amount {
		return false
	}
	if a.Txn.Height != b.Txn.Height {
		return false
	}
	if !bytes.Equal(a.Txn.Signature[:], b.Txn.Signature[:]) {
		return false
	}
	if !bytes.Equal(a.Txn.PublicKey[:], b.Txn.PublicKey[:]) {
		return false
	}
	if !bytes.Equal(a.Signature[:], b.Signature[:]) {
		return false
	}
	if !bytes.Equal(a.PublicKey[:], b.PublicKey[:]) {
		return false
	}
	if !bytes.Equal(a.Proof[:], b.Proof[:]) {
		return false
	}
	return true
}
