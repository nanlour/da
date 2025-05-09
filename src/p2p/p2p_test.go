package p2p

import (
	"sync"
	"testing"
	"time"

	"github.com/nanlour/da/src/block"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBlockchain is a mock implementation of BlockchainInterface for testing
type MockBlockchain struct {
	blocks      map[[32]byte]*block.Block
	tipHash     [32]byte
	tipHeight   int64
	blocksMutex sync.RWMutex
}

func NewMockBlockchain() *MockBlockchain {
	return &MockBlockchain{
		blocks:    make(map[[32]byte]*block.Block),
		tipHeight: -1,
	}
}

func (m *MockBlockchain) AddBlock(b *block.Block) error {
	hash := b.Hash()

	m.blocksMutex.Lock()
	defer m.blocksMutex.Unlock()

	m.blocks[hash] = b
	if int64(b.Height) > m.tipHeight {
		m.tipHeight = int64(b.Height)
		m.tipHash = hash
	}
	return nil
}

func (m *MockBlockchain) AddTxn(b *block.Transaction) error {
	return nil
}

func (m *MockBlockchain) GetBlockByHash(hash []byte) (*block.Block, error) {
	m.blocksMutex.RLock()
	defer m.blocksMutex.RUnlock()

	var hashArray [32]byte
	copy(hashArray[:], hash)

	if block, exists := m.blocks[hashArray]; exists {
		return block, nil
	}
	return nil, nil
}

func (m *MockBlockchain) GetTipBlock() (*block.Block, error) {
	m.blocksMutex.RLock()
	defer m.blocksMutex.RUnlock()

	return m.GetBlockByHash(m.tipHash[:])
}

func (m *MockBlockchain) GetBlockHeight(hash []byte) (int64, error) {
	m.blocksMutex.RLock()
	defer m.blocksMutex.RUnlock()

	var hashArray [32]byte
	copy(hashArray[:], hash)

	if block, exists := m.blocks[hashArray]; exists {
		return int64(block.Height), nil
	}
	return 0, nil
}

// TestServiceCreation tests creating, starting, and stopping a P2P service
func TestServiceCreation(t *testing.T) {
	// Create a mock blockchain
	mockBC := NewMockBlockchain()

	// Create a P2P service
	service, err := NewService("/ip4/127.0.0.1/tcp/0", mockBC)
	require.NoError(t, err)
	require.NotNil(t, service)

	// Start the service
	err = service.Start()
	require.NoError(t, err)

	// Verify service is running
	assert.NotEmpty(t, service.host.Addrs())

	// Stop the service
	err = service.Stop()
	require.NoError(t, err)
}

// TestPeerConnection tests connecting two P2P nodes
func TestPeerConnection(t *testing.T) {
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

	// Get multiaddress of service2 to connect from service1
	addr2 := service2.host.Addrs()[0].String() + "/p2p/" + service2.host.ID().String()

	// Connect service1 to service2
	err = service1.Connect(addr2)
	require.NoError(t, err)

	// Check if peers are connected
	time.Sleep(100 * time.Millisecond) // Give some time for connection to establish

	peers := service1.Peers()
	assert.Contains(t, peers, service2.host.ID())
}

// TestProtocolHandlers tests the custom protocol handlers (GetBlockByHash and GetTip)
func TestProtocolHandlers(t *testing.T) {
	// Create two mock blockchains
	mockBC1 := NewMockBlockchain()
	mockBC2 := NewMockBlockchain()

	// Create a test block and add it to mockBC2
	testBlock := &block.Block{
		Height: 1,
		Txn: block.Transaction{
			Amount: 100,
		},
	}
	mockBC2.AddBlock(testBlock)
	testBlockHash := testBlock.Hash()

	testBlock2 := &block.Block{
		Height: 2,
		Txn: block.Transaction{
			Amount: 101,
		},
	}
	mockBC2.AddBlock(testBlock2)

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

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Test GetBlockByHash
	retrievedBlock, err := service1.GetBlockByHash(testBlockHash, service2.host.ID())
	require.NoError(t, err)
	assert.NotNil(t, retrievedBlock)
	assert.Equal(t, testBlock.Height, retrievedBlock.Height)
	assert.Equal(t, testBlock.Txn.Amount, retrievedBlock.Txn.Amount)

	// Test GetTip
	retrievedBlock, err = service1.GetTip(service2.host.ID())
	require.NoError(t, err)
	assert.NotNil(t, retrievedBlock)
	assert.Equal(t, testBlock2.Height, retrievedBlock.Height)
	assert.Equal(t, testBlock2.Txn.Amount, retrievedBlock.Txn.Amount)
}

// TestDiscovery tests peer discovery mechanisms
func TestDiscovery(t *testing.T) {
	// Create multiple services with mock blockchains
	services := make([]*Service, 3)
	mockBCs := make([]*MockBlockchain, 3)

	for i := 0; i < 3; i++ {
		mockBCs[i] = NewMockBlockchain()
		var err error
		services[i], err = NewService("/ip4/127.0.0.1/tcp/0", mockBCs[i])
		require.NoError(t, err)
	}

	// Start all services
	for i, service := range services {
		err := service.Start()
		require.NoError(t, err, "Failed to start service %d", i)
		defer service.Stop()
	}

	// Connect service[0] to service[1]
	addr1 := services[1].host.Addrs()[0].String() + "/p2p/" + services[1].host.ID().String()
	err := services[0].Connect(addr1)
	require.NoError(t, err)

	// Connect service[1] to service[2]
	addr2 := services[2].host.Addrs()[0].String() + "/p2p/" + services[2].host.ID().String()
	err = services[1].Connect(addr2)
	require.NoError(t, err)

	// Wait for DHT discovery to propagate
	time.Sleep(2 * time.Second)

	// Eventually service[0] should discover service[2] through DHT
	discovered := false
	for i := 0; i < 5; i++ { // Try a few times
		peers := services[0].Peers()
		for _, p := range peers {
			if p == services[2].host.ID() {
				discovered = true
				break
			}
		}
		if discovered {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	assert.True(t, discovered, "Service[0] should eventually discover Service[2] through DHT")
}
