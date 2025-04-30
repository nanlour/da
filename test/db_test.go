package test

import (
	"os"
	"testing"

	"github.com/nanlour/da/util"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestDBManager(t *testing.T) {
	// Temporary directory for testing
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir) // Clean up after test

	// Initialize the database
	err := util.InitialDB(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	defer util.MainDB.Close()

	// Test Insert
	key := []byte("testKey")
	value := []byte("testValue")
	err = util.MainDB.Insert(key, value)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}

	// Test Get
	retrievedValue, err := util.MainDB.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if string(retrievedValue) != string(value) {
		t.Errorf("Expected value %s, got %s", value, retrievedValue)
	}

	// Test BatchInsert
	batch := new(leveldb.Batch)
	batch.Put([]byte("batchKey1"), []byte("batchValue1"))
	batch.Put([]byte("batchKey2"), []byte("batchValue2"))
	err = util.MainDB.BatchInsert(batch)
	if err != nil {
		t.Errorf("BatchInsert failed: %v", err)
	}

	// Verify BatchInsert
	retrievedBatchValue, err := util.MainDB.Get([]byte("batchKey1"))
	if err != nil || string(retrievedBatchValue) != "batchValue1" {
		t.Errorf("BatchInsert verification failed for batchKey1")
	}
}

func TestFileOperations(t *testing.T) {
	// Example test for file-related operations
	filePath := "./testfile.txt"
	defer os.Remove(filePath) // Clean up after test

	// Write to file
	content := []byte("Hello, World!")
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write to file: %v", err)
	}

	// Read from file
	readContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read from file: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("Expected content %s, got %s", content, readContent)
	}
}
