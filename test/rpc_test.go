package test

import (
	"fmt"
	"net/rpc"
	"testing"
	"time"

	"github.com/nanlour/da/block"
	daRPC "github.com/nanlour/da/rpc"
	"github.com/nanlour/da/util"
)

// TestRPCServer tests the basic functionality of the RPC server
func TestRPCServer(t *testing.T) {
	// Create a new RPC server on an arbitrary port
	port := 9876
	server := daRPC.NewRPCServer(port)

	// Test server start
	err := server.Start()
	if err != nil {
		t.Fatalf("Failed to start RPC server: %v", err)
	}

	// Give the server a moment to initialize
	time.Sleep(100 * time.Millisecond)

	// Test starting an already running server (should fail)
	err = server.Start()
	if err == nil {
		t.Errorf("Expected error when starting already running server, got nil")
	}

	// Test client connection to server
	client, err := rpc.Dial("tcp", "localhost:9876")
	if err != nil {
		t.Fatalf("Failed to connect to RPC server: %v", err)
	}
	defer client.Close()

	// Test server stop
	err = server.Stop()
	if err != nil {
		t.Fatalf("Failed to stop RPC server: %v", err)
	}

	// Test stopping an already stopped server (should fail)
	err = server.Stop()
	if err == nil {
		t.Errorf("Expected error when stopping already stopped server, got nil")
	}
}

// TestBlockchainService tests the functionality of the BlockchainService RPC methods
func TestBlockchainService(t *testing.T) {
	// Initialize a test database in a temporary directory
	tempDBPath := t.TempDir() + "/testdb"
	err := util.InitialDB(tempDBPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer util.MainDB.Close()

	// Create test data
	testBlockHash := make([]byte, 32)
	copy(testBlockHash, []byte("testblockhash12345678901234567890"))

	testAddr := make([]byte, 32)
	copy(testAddr, []byte("testaddress12345678901234567890"))

	// Prepare test data in database
	testHeight := int64(123)
	err = util.MainDB.InsertBlockHeight(testBlockHash, testHeight)
	if err != nil {
		t.Fatalf("Failed to insert test block height: %v", err)
	}

	err = util.MainDB.InsertTipHash(testBlockHash)
	if err != nil {
		t.Fatalf("Failed to insert test tip hash: %v", err)
	}

	testBalance := 100.5
	err = util.MainDB.InsertAccountBalance(testAddr, testBalance)
	if err != nil {
		t.Fatalf("Failed to insert test account balance: %v", err)
	}

	// Create a test block
	var testBlockData block.Block
	testBlockData.PreHash = [32]byte{1, 2, 3}
	testBlockData.EpochBeginHash = [32]byte{4, 5, 6}
	testBlockData.PublicKey = [64]byte{7, 8, 9}
	testBlockData.Signature = [64]byte{10, 11, 12}
	testBlockData.Proof = [516]byte{13, 14, 15}

	// Insert the test block
	err = util.MainDB.InsertHashBlock(testBlockHash, &testBlockData)
	if err != nil {
		t.Fatalf("Failed to insert test block: %v", err)
	}

	// Start an RPC server for testing
	port := 9877
	server := daRPC.NewRPCServer(port)

	err = server.Start()
	if err != nil {
		t.Fatalf("Failed to start RPC server: %v", err)
	}
	defer server.Stop()

	// Allow server to initialize
	time.Sleep(100 * time.Millisecond)

	// Connect a client to the RPC server
	client, err := rpc.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("Failed to connect to RPC server: %v", err)
	}
	defer client.Close()

	// Test 1: GetTip RPC call
	t.Run("GetTip", func(t *testing.T) {
		var tip daRPC.Tip
		err = client.Call("BlockchainService.GetTip", &struct{}{}, &tip)
		if err != nil {
			t.Errorf("Error calling GetTip: %v", err)
		}

		var expectedHash [32]byte
		copy(expectedHash[:], testBlockHash)

		if tip.Hash != expectedHash {
			t.Errorf("GetTip returned wrong hash: got %x, want %x", tip.Hash, expectedHash)
		}
		if tip.Height != uint64(testHeight) {
			t.Errorf("GetTip returned wrong height: got %d, want %d", tip.Height, testHeight)
		}
	})

	// Test 2: GetBalanceByAddress RPC call
	t.Run("GetBalanceByAddress", func(t *testing.T) {
		var balance float64
		var addrArray [32]byte
		copy(addrArray[:], testAddr)

		err = client.Call("BlockchainService.GetBalanceByAddress", addrArray, &balance)
		if err != nil {
			t.Errorf("Error calling GetBalanceByAddress: %v", err)
		}
		if balance != testBalance {
			t.Errorf("GetBalanceByAddress returned wrong balance: got %f, want %f", balance, testBalance)
		}
	})

	// Test 3: GetBlockByHash RPC call
	t.Run("GetBlockByHash", func(t *testing.T) {
		var blockData block.Block
		var hashArray [32]byte
		copy(hashArray[:], testBlockHash)

		err = client.Call("BlockchainService.GetBlockByHash", hashArray, &blockData)
		if err != nil {
			t.Errorf("Error calling GetBlockByHash: %v", err)
		}

		if blockData.PreHash != testBlockData.PreHash ||
			blockData.EpochBeginHash != testBlockData.EpochBeginHash ||
			blockData.PublicKey != testBlockData.PublicKey ||
			blockData.Signature != testBlockData.Signature {
			t.Errorf("GetBlockByHash returned incorrect block data")
		}
	})

	// Test 4: GetBlockByHash with non-existent hash (expecting error)
	t.Run("GetBlockByHash with non-existent hash", func(t *testing.T) {
		var blockData block.Block
		nonExistentHash := [32]byte{99, 99, 99}

		err = client.Call("BlockchainService.GetBlockByHash", nonExistentHash, &blockData)
		if err == nil {
			t.Errorf("Expected error for non-existent block hash, got nil")
		}
	})
}
