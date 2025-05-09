package rpc

import (
	"errors"
	"net/rpc"
	"testing"
	"time"

	"github.com/nanlour/da/src/block"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBlockchain implements the BlockchainInterface for testing
type MockBlockchain struct {
	tipBlock      *block.Block
	blocks        map[[32]byte]*block.Block
	balances      map[[32]byte]float64
	sendTxnCalled bool
	sendTxnError  error
}

// NewMockBlockchain creates a new mock blockchain for testing
func NewMockBlockchain() *MockBlockchain {
	// Create a test transaction
	var txn block.Transaction
	txn.FromAddress = [32]byte{1, 2, 3}
	txn.ToAddress = [32]byte{4, 5, 6}
	txn.Amount = 100.0
	txn.Height = 1

	// Create a test block
	var tipBlock block.Block
	tipBlock.Height = 1
	tipBlock.Txn = txn

	tipHash := tipBlock.Hash()

	// Initialize with some test data
	blocks := make(map[[32]byte]*block.Block)
	blocks[tipHash] = &tipBlock

	balances := make(map[[32]byte]float64)
	balances[[32]byte{1, 2, 3}] = 500.0
	balances[[32]byte{4, 5, 6}] = 200.0

	return &MockBlockchain{
		tipBlock: &tipBlock,
		blocks:   blocks,
		balances: balances,
	}
}

// GetBlockByHash implements BlockchainInterface
func (m *MockBlockchain) GetBlockByHash(hash []byte) (*block.Block, error) {
	var hashArray [32]byte
	copy(hashArray[:], hash)

	if block, exists := m.blocks[hashArray]; exists {
		return block, nil
	}
	return nil, errors.New("block not found")
}

// GetTipBlock implements BlockchainInterface
func (m *MockBlockchain) GetTipBlock() (*block.Block, error) {
	if m.tipBlock == nil {
		return nil, errors.New("no tip block")
	}
	return m.tipBlock, nil
}

// GetAddress implements BlockchainInterface
func (m *MockBlockchain) GetAddress() ([32]byte, error) {
	return [32]byte{1, 2, 3}, nil
}

// GetAccountBalance implements BlockchainInterface
func (m *MockBlockchain) GetAccountBalance(address *[32]byte) (float64, error) {
	if balance, exists := m.balances[*address]; exists {
		return balance, nil
	}
	return 0, errors.New("account not found")
}

// SendTxn implements BlockchainInterface
func (m *MockBlockchain) SendTxn(dest [32]byte, amount float64) error {
	m.sendTxnCalled = true
	// Return pre-configured error or nil
	return m.sendTxnError
}

// Helper method to configure SendTxn to return an error
func (m *MockBlockchain) SetSendTxnError(err error) {
	m.sendTxnError = err
}

// TestStartStopRPCServer tests starting and stopping the RPC server
func TestStartStopRPCServer(t *testing.T) {
	// Create mock blockchain
	mockBC := NewMockBlockchain()

	// Create RPC server with a random port
	server := NewRPCServer(0)

	// Start the server
	err := server.Start(mockBC)
	require.NoError(t, err, "Failed to start RPC server")

	// Get the dynamically assigned port
	addr := server.listener.Addr().String()

	// Try to connect to the server
	client, err := rpc.Dial("tcp", addr)
	require.NoError(t, err, "Failed to connect to RPC server")

	// Close the client connection
	client.Close()

	// Stop the server
	err = server.Stop()
	require.NoError(t, err, "Failed to stop RPC server")

	// Try to connect again, should fail
	_, err = rpc.Dial("tcp", addr)
	assert.Error(t, err, "Should not be able to connect after server is stopped")
}

// TestGetTip tests the GetTip RPC method
func TestGetTip(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Call the GetTip method
	var reply [32]byte
	err := client.Call("BlockchainService.GetTip", struct{}{}, &reply)
	require.NoError(t, err, "GetTip RPC call failed")

	// Verify the result matches the tip block hash
	expectedHash := mockBC.tipBlock.Hash()
	assert.Equal(t, expectedHash, reply, "GetTip returned incorrect hash")
}

// TestGetBlockByHash tests the GetBlockByHash RPC method
func TestGetBlockByHash(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Get the hash of a known block
	knownHash := mockBC.tipBlock.Hash()

	// Call the GetBlockByHash method
	var reply block.Block
	err := client.Call("BlockchainService.GetBlockByHash", knownHash, &reply)
	require.NoError(t, err, "GetBlockByHash RPC call failed")

	// Verify the returned block matches the expected block
	assert.Equal(t, mockBC.tipBlock.Height, reply.Height, "Block height does not match")
	assert.Equal(t, mockBC.tipBlock.Txn.Amount, reply.Txn.Amount, "Transaction amount does not match")
}

// TestGetBlockByHashNotFound tests the GetBlockByHash RPC method with a non-existent block
func TestGetBlockByHashNotFound(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Create a hash for a non-existent block
	var nonExistentHash [32]byte
	for i := 0; i < 32; i++ {
		nonExistentHash[i] = byte(i + 100)
	}

	// Call the GetBlockByHash method
	var reply block.Block
	err := client.Call("BlockchainService.GetBlockByHash", nonExistentHash, &reply)
	assert.Error(t, err, "GetBlockByHash should fail for non-existent block")
	assert.Contains(t, err.Error(), "block not found", "Error message should indicate block not found")
}

// TestGetBalanceByAddress tests the GetBalanceByAddress RPC method
func TestGetBalanceByAddress(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Get balance for an address with known balance
	address := [32]byte{1, 2, 3}
	expectedBalance := mockBC.balances[address]

	// Call the GetBalanceByAddress method
	var reply float64
	err := client.Call("BlockchainService.GetBalanceByAddress", address, &reply)
	require.NoError(t, err, "GetBalanceByAddress RPC call failed")

	// Verify the returned balance matches the expected balance
	assert.Equal(t, expectedBalance, reply, "Returned balance does not match expected value")
}

// TestGetBalanceByAddressNotFound tests the GetBalanceByAddress RPC method with a non-existent address
func TestGetBalanceByAddressNotFound(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Create a non-existent address
	var nonExistentAddr [32]byte
	for i := 0; i < 32; i++ {
		nonExistentAddr[i] = byte(i + 200)
	}

	// Call the GetBalanceByAddress method
	var reply float64
	err := client.Call("BlockchainService.GetBalanceByAddress", nonExistentAddr, &reply)
	assert.Error(t, err, "GetBalanceByAddress should fail for non-existent address")
	assert.Contains(t, err.Error(), "account not found", "Error message should indicate account not found")
}

// TestSendTxn tests the SendTxn RPC method
func TestSendTxn(t *testing.T) {
	mockBC := NewMockBlockchain()
	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Create transaction arguments
	args := SendTxnArgs{
		Destination: [32]byte{7, 8, 9},
		Amount:      50.0,
	}

	// Call the SendTxn method
	var reply bool
	err := client.Call("BlockchainService.SendTxn", &args, &reply)
	require.NoError(t, err, "SendTxn RPC call failed")

	// Verify SendTxn was called on the mock blockchain
	assert.True(t, mockBC.sendTxnCalled, "SendTxn was not called on the blockchain")
	assert.True(t, reply, "SendTxn should return true on success")
}

// TestSendTxnError tests the SendTxn RPC method when the blockchain returns an error
func TestSendTxnError(t *testing.T) {
	mockBC := NewMockBlockchain()
	mockBC.SetSendTxnError(errors.New("insufficient funds"))

	server, client := setupRPCTest(t, mockBC)
	defer server.Stop()

	// Create transaction arguments
	args := SendTxnArgs{
		Destination: [32]byte{7, 8, 9},
		Amount:      5000.0, // More than available balance
	}

	// Call the SendTxn method
	var reply bool
	err := client.Call("BlockchainService.SendTxn", &args, &reply)
	assert.Error(t, err, "SendTxn should fail when blockchain returns an error")
	assert.Contains(t, err.Error(), "insufficient funds", "Error message should indicate insufficient funds")
}

// Helper function to set up RPC server and client for tests
func setupRPCTest(t *testing.T, mockBC *MockBlockchain) (*RPCServer, *rpc.Client) {
	// Create RPC server with a random port
	server := NewRPCServer(0)

	// Start the server
	err := server.Start(mockBC)
	require.NoError(t, err, "Failed to start RPC server")

	// Get the dynamically assigned port
	addr := server.listener.Addr().String()

	// Connect a client
	client, err := rpc.Dial("tcp", addr)
	require.NoError(t, err, "Failed to connect to RPC server")

	// Allow time for connection to establish
	time.Sleep(50 * time.Millisecond)

	return server, client
}
