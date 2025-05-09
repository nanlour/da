package test

import (
	"os"
	"testing"

	"github.com/nanlour/da/db"
)

func TestDBManager(t *testing.T) {
	// Temporary directory for testing
	tempDir := t.TempDir()
	defer os.RemoveAll(tempDir) // Clean up after test

	// Initialize the database
	err := db.InitialDB(tempDir)
	if err != nil {
		t.Fatalf("Failed to initialize DB: %v", err)
	}
	defer db.MainDB.Close()

	// Test Insert
	key := []byte("testKey")
	value := []byte("testValue")
	err = db.MainDB.Insert(key, value)
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}

	// Test Get
	retrievedValue, err := db.MainDB.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if string(retrievedValue) != string(value) {
		t.Errorf("Expected value %s, got %s", value, retrievedValue)
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
