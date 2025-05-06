package p2p

import (
	"testing"
	"time"

	"github.com/nanlour/da/block"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPubSub tests the PubSub functionality for broadcasting blocks and transactions
func TestPubSub(t *testing.T) {
	// Create two mock blockchains
	mockBC1 := NewMockBlockchain()
	mockBC2 := NewMockBlockchain()

	// Create two P2P services
	service1, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC1)
	require.NoError(t, err)

	service2, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC2)
	require.NoError(t, err)

	// Start both services
	err = service1.Start()
	require.NoError(t, err)
	defer service1.Stop()

	err = service2.Start()
	require.NoError(t, err)
	defer service2.Stop()

	// Connect service1 to service2
	addr2 := service2.host.Addrs()[0].String() + "/p2p/" + service2.host.ID().String()
	err = service1.Connect(addr2)
	require.NoError(t, err)

	// Wait for connection to establish and PubSub to initialize
	time.Sleep(500 * time.Millisecond)

	// Test Block Broadcasting
	t.Run("Block Broadcasting", func(t *testing.T) {
		// Create a test block
		testBlock := &block.Block{
			Height: 1,
			Txn: block.Transaction{
				Amount: 100,
			},
		}

		// Get initial blockchain state
		initialBlockCount := len(mockBC2.blocks)

		// Broadcast the block from service1
		err = service1.BroadcastBlock(testBlock)
		require.NoError(t, err)

		// Wait for the block to be processed
		time.Sleep(500 * time.Millisecond)

		// Verify that service2 received and added the block
		mockBC2.blocksMutex.RLock()
		newBlockCount := len(mockBC2.blocks)
		mockBC2.blocksMutex.RUnlock()

		assert.Greater(t, newBlockCount, initialBlockCount, "Block should be added to the second blockchain")

		// Verify block exists in mockBC2
		blockHash := testBlock.Hash()
		block, err := mockBC2.GetBlockByHash(blockHash[:])
		assert.NoError(t, err)
		assert.NotNil(t, block)
		assert.Equal(t, testBlock.Height, block.Height)
		assert.Equal(t, testBlock.Txn.Amount, block.Txn.Amount)
	})

	// Test Transaction Broadcasting
	t.Run("Transaction Broadcasting", func(t *testing.T) {
		// Create a test transaction
		testTx := &Transaction{
			Data: []byte("test transaction data"),
		}

		// Broadcast the transaction from service2
		err = service2.BroadcastTransaction(testTx)
		require.NoError(t, err)

		// Wait for the transaction to be processed
		// Since we don't have a way to check if the transaction was received
		// (transactions are just logged, not stored), we just check that
		// broadcasting doesn't error
		time.Sleep(500 * time.Millisecond)

		// The test passes if no error occurs during broadcasting
		assert.NoError(t, err)
	})
}

// TestPubSubWithMultiplePeers tests PubSub functionality with multiple peers
func TestPubSubWithMultiplePeers(t *testing.T) {
	// Create three mock blockchains
	mockBC1 := NewMockBlockchain()
	mockBC2 := NewMockBlockchain()
	mockBC3 := NewMockBlockchain()

	// Create three P2P services
	service1, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC1)
	require.NoError(t, err)

	service2, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC2)
	require.NoError(t, err)

	service3, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC3)
	require.NoError(t, err)

	// Start all services
	err = service1.Start()
	require.NoError(t, err)
	defer service1.Stop()

	err = service2.Start()
	require.NoError(t, err)
	defer service2.Stop()

	err = service3.Start()
	require.NoError(t, err)
	defer service3.Stop()

	// Connect service1 to service2
	addr2 := service2.host.Addrs()[0].String() + "/p2p/" + service2.host.ID().String()
	err = service1.Connect(addr2)
	require.NoError(t, err)

	// Connect service2 to service3
	addr3 := service3.host.Addrs()[0].String() + "/p2p/" + service3.host.ID().String()
	err = service2.Connect(addr3)
	require.NoError(t, err)

	// Wait for connections to establish and PubSub to initialize
	time.Sleep(1 * time.Second)

	// Create a test block
	testBlock := &block.Block{
		Height: 5,
		Txn: block.Transaction{
			Amount: 200,
		},
	}

	// Broadcast the block from service1
	err = service1.BroadcastBlock(testBlock)
	require.NoError(t, err)

	// Wait for the block to propagate through the network
	time.Sleep(1 * time.Second)

	// Verify that both service2 and service3 received and added the block
	blockHash := testBlock.Hash()

	// Check service2
	block2, err := mockBC2.GetBlockByHash(blockHash[:])
	assert.NoError(t, err)
	assert.NotNil(t, block2)
	assert.Equal(t, testBlock.Height, block2.Height)

	// Check service3
	block3, err := mockBC3.GetBlockByHash(blockHash[:])
	assert.NoError(t, err)
	assert.NotNil(t, block3)
	assert.Equal(t, testBlock.Height, block3.Height)
}
