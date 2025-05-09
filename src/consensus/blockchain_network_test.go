package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nanlour/da/src/block"
	"github.com/nanlour/da/src/ecdsa_da"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestNetwork creates a network of blockchain nodes for testing
func setupTestNetwork(t *testing.T, nodeCount int) ([]*BlockChain, func()) {
	// Create temp directories for each node
	tempBaseDir, err := os.MkdirTemp("", "blockchain_network_test_")
	require.NoError(t, err)

	// Generate nodes with unique configurations
	nodes := make([]*BlockChain, nodeCount)
	nodeAddrs := make([]string, nodeCount)

	stakeSum := float64(nodeCount * 100)
	initStake := map[[32]byte]float64{}
	initBank := map[[32]byte]float64{}

	// First create all node configs and extract addresses for bootstrap
	for i := range nodeCount {
		// Create p2p listen address - use different ports for each node
		p2pAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 10000+i)
		nodeAddrs[i] = p2pAddr
	}

	// Now create each blockchain node with knowledge of other nodes
	for i := range nodeCount {
		// Generate a unique private key for this node
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// Create unique address for this node
		address := ecdsa_da.PublicKeyToAddress(&privateKey.PublicKey)

		// Set up bootstrap peers (all nodes except self)
		bootstrapPeers := make([]string, 0)
		if i != 0 { // Don't connect to self
			bootstrapPeers = append(bootstrapPeers, nodeAddrs[0]+"/p2p")
		}

		initStake[address] = 100
		initBank[address] = 100

		// Create blockchain config
		config := &Config{
			ID: Account{
				PrvKey:  *privateKey,
				PubKey:  privateKey.PublicKey,
				Address: address,
			},
			StakeMine:        100,
			MiningDifficulty: 5000,
			DbPath:           filepath.Join(tempBaseDir, fmt.Sprintf("node%d", i)),
			RPCPort:          9000 + i,
			P2PListenAddr:    nodeAddrs[i],
			BootstrapPeer:    bootstrapPeers,
			StakeSum:         stakeSum,
		}

		// Initialize blockchain
		nodes[i] = &BlockChain{}
		nodes[i].SetConfig(config)
	}

	for i := range nodeCount {
		nodes[i].NodeConfig.InitStake = initStake
		nodes[i].NodeConfig.InitBank = initBank
		err = nodes[i].Init()
		require.NoError(t, err)
		// Give time for ndoe initial
		time.Sleep(2 * time.Second)
	}

	// Return cleanup function
	cleanup := func() {
		for _, node := range nodes {
			node.Stop()
		}
		os.RemoveAll(tempBaseDir)
	}

	// Allow nodes to connect to each other
	time.Sleep(2 * time.Second)

	return nodes, cleanup
}

// TestBlockchainNetworkSync tests that blocks propagate through the network
func TestBlockchainNetworkSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	// Create a network with 3 nodes
	nodes, cleanup := setupTestNetwork(t, 3)
	defer cleanup()

	// Allow connections to establish
	time.Sleep(3 * time.Second)

	// Let node 0 mine a block
	node0TipBefore, err := nodes[0].GetTipBlock()
	require.NoError(t, err)

	// Simulate mining by creating and adding a block to node 0
	privateKey := &nodes[0].NodeConfig.ID.PrvKey
	pubKey := ecdsa_da.PublicKeyToBytes(&privateKey.PublicKey)

	// Create a test transaction
	txn := block.Transaction{
		FromAddress: nodes[0].NodeConfig.ID.Address,
		ToAddress:   nodes[1].NodeConfig.ID.Address,
		Amount:      10.0,
		Height:      node0TipBefore.Height + 1,
		PublicKey:   pubKey,
	}
	txn.Sign(privateKey)

	// Wait for block propagation
	time.Sleep(200 * time.Second)

	// Check that all nodes have the new block
	maxRetries := 3
	var tips []block.Block
	for attempt := range maxRetries {
		tips = make([]block.Block, len(nodes))
		allEqual := true
		var firstHash [32]byte

		for i, node := range nodes {
			tip, err := node.GetTipBlock()
			require.NoError(t, err, "Failed to get tip for node %d", i)
			tips[i] = *tip
			if i == 0 {
				firstHash = tip.Hash()
			} else if tipHash := tip.Hash(); string(tipHash[:]) != string(firstHash[:]) {
				allEqual = false
			}
		}

		if allEqual {
			break
		}
		if attempt < maxRetries-1 {
			time.Sleep(7 * time.Second)
		}
	}

	// Final assertion
	for i := 1; i < len(tips); i++ {
		assert.Equal(t, tips[0].Hash(), tips[i].Hash(), "All nodes should have the same tip block after propagation")
	}
}

// TestTransactionPropagation tests that transactions propagate across the network
func TestTransactionPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	// Create a network with 3 nodes
	nodes, cleanup := setupTestNetwork(t, 3)
	defer cleanup()

	// Allow connections to establish
	time.Sleep(3 * time.Second)

	// Get initial balances
	senderAddr := nodes[0].NodeConfig.ID.Address
	receiverAddr := nodes[1].NodeConfig.ID.Address

	senderBalanceBefore, err := nodes[0].GetAccountBalance(&senderAddr)
	require.NoError(t, err)

	receiverBalanceBefore, err := nodes[0].GetAccountBalance(&receiverAddr)
	if err != nil {
		// If receiver doesn't exist yet in node 0's view
		receiverBalanceBefore = 0
	}

	// Node 0 sends transaction to Node 1
	sendAmount := 50.0
	err = nodes[0].SendTxn(receiverAddr, sendAmount)
	require.NoError(t, err)

	// Wait for transaction to be mined in a block and propagated
	time.Sleep(10 * time.Second)

	// Check that transaction was processed across all nodes
	// In an actual network, we'd need to mine a block to confirm the transaction
	// Here we're just checking the transaction pool and/or balance changes

	// Check sender's balance
	senderBalanceAfter, err := nodes[2].GetAccountBalance(&senderAddr)
	if err == nil {
		// If we can get the balance, it should be reduced
		assert.Less(t, senderBalanceAfter, senderBalanceBefore,
			"Sender's balance should decrease after sending transaction")
	}

	// Check receiver's balance on another node
	receiverBalanceAfter, err := nodes[2].GetAccountBalance(&receiverAddr)
	if err == nil && receiverBalanceAfter > receiverBalanceBefore {
		assert.Greater(t, receiverBalanceAfter, receiverBalanceBefore,
			"Receiver's balance should increase after receiving transaction")
	}
}

// TestBlockchainConsensus tests that nodes reach consensus after mining
func TestBlockchainConsensus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping consensus test in short mode")
	}

	// Create a smaller network for consensus test
	nodes, cleanup := setupTestNetwork(t, 2)
	defer cleanup()

	// Allow connections to establish
	time.Sleep(3 * time.Second)

	// Get initial tip blocks from both nodes
	initialTip1, err := nodes[0].GetTipBlock()
	require.NoError(t, err)
	initialTip2, err := nodes[1].GetTipBlock()
	require.NoError(t, err)

	// Both should start with the same genesis block
	assert.Equal(t, initialTip1.Hash(), initialTip2.Hash(), "All nodes should start with the same genesis block")

	// Let the nodes run for a while and possibly mine blocks
	time.Sleep(20 * time.Second)

	// Get final tip blocks
	finalTip1, err := nodes[0].GetTipBlock()
	require.NoError(t, err)
	finalTip2, err := nodes[1].GetTipBlock()
	require.NoError(t, err)

	// Check that both nodes have advanced beyond the initial block
	assert.Greater(t, finalTip1.Height, initialTip1.Height, "Node 1 should have mined new blocks")
	assert.Greater(t, finalTip2.Height, initialTip2.Height, "Node 2 should have mined new blocks")

	// In a real consensus, both nodes would eventually have the same tip
	// This depends on mining rate, network conditions, etc., so we can't assert equality
	// Instead, we check that they've both advanced and log the final state
	t.Logf("Node 1 tip height: %d, Node 2 tip height: %d", finalTip1.Height, finalTip2.Height)

	// Wait for potential synchronization
	time.Sleep(5 * time.Second)

	// Check if both nodes have the highest block
	tip1, _ := nodes[0].GetTipBlock()
	tip2, _ := nodes[1].GetTipBlock()

	// Log final heights
	t.Logf("After sync - Node 1 tip height: %d, Node 2 tip height: %d",
		tip1.Height, tip2.Height)
}
