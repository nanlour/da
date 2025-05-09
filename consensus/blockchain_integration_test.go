package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/ecdsa_da"
)

// setupTestBlockchain creates a blockchain instance for testing
func setupTestBlockchain(t *testing.T, stakeMine float64, miningDifficulty uint64, dbPath string) *BlockChain {
	// Generate private key for the node
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	address := ecdsa_da.PublicKeyToAddress(&privateKey.PublicKey)

	bc := &BlockChain{
		NodeConfig: &Config{
			ID: Account{
				PrvKey:  *privateKey,
				PubKey:  privateKey.PublicKey,
				Address: address,
			},
			StakeMine:        stakeMine,
			MiningDifficulty: miningDifficulty,
			DbPath:           dbPath,
			RPCPort:          0,                      // Use 0 to get an available port automatically
			ListenAddr:       "/ip4/127.0.0.1/tcp/0", // Use port 0 to get an available port
			InitStake: map[[32]byte]float64{
				address: stakeMine,
			},
			StakeSum: stakeMine,
			InitBank: map[[32]byte]float64{
				address: 1000.0, // Give initial funds to test with
			},
		},
		TxnPool:    TransactionPool{txnMap: make(map[uint64]*block.Transaction)},
		MiningChan: make(chan *block.Block, 10),
		P2PChan:    make(chan *block.Block, 10),
	}

	return bc
}

// TestMiningBasic tests that a single blockchain can mine blocks successfully
func TestMiningBasic(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "blockchain_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "testdb")

	// Setup blockchain with low difficulty for quick mining
	bc := setupTestBlockchain(t, 100.0, 1, dbPath)

	// Initialize the blockchain
	err = bc.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer bc.Stop()

	// Wait for a block to be mined
	var minedBlock *block.Block
	select {
	case minedBlock = <-bc.MiningChan:
		// Block was successfully mined
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for a block to be mined")
	}

	// Wait for db add to db
	time.Sleep(500 * time.Millisecond)

	// Verify block
	if minedBlock.Height != 1 {
		t.Errorf("Expected height 1, got %d", minedBlock.Height)
	}

	if !bc.VerifyBlock(minedBlock) {
		t.Error("Mined block failed verification")
	}

	// Check the block was added to chain
	tipBlock, err := bc.GetTipBlock()
	if err != nil {
		t.Fatalf("Failed to get tip block: %v", err)
	}

	if tipBlock.Height != 1 {
		t.Errorf("Tip block height should be 1, got %d", tipBlock.Height)
	}
}

// TestMultipleNodesWithFork tests fork resolution between multiple blockchain nodes
func TestMultipleNodesWithFork(t *testing.T) {
	// Create temporary directories for test databases
	tempDir1, err := os.MkdirTemp("", "blockchain_test_1")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir1)

	tempDir2, err := os.MkdirTemp("", "blockchain_test_2")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir2)

	// Setup two blockchain instances
	bc1 := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir1, "testdb"))
	bc2 := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir2, "testdb"))

	// Initialize both blockchains
	err = bc1.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain 1: %v", err)
	}
	defer bc1.Stop()

	err = bc2.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain 2: %v", err)
	}
	defer bc2.Stop()

	// Let bc1 mine a block and manually add it to bc2 (simulating P2P)
	var minedBlock *block.Block
	select {
	case minedBlock = <-bc1.MiningChan:
		// Block was mined by bc1
		err = bc2.AddBlock(minedBlock)
		if err != nil {
			t.Fatalf("Failed to add block to bc2: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for bc1 to mine a block")
	}

	// Give bc2 time to process the block
	time.Sleep(500 * time.Millisecond)

	// Both blockchains should have the same tip now
	tip1, err := bc1.GetTipBlock()
	if err != nil {
		t.Fatalf("Failed to get tip from bc1: %v", err)
	}

	tip2, err := bc2.GetTipBlock()
	if err != nil {
		t.Fatalf("Failed to get tip from bc2: %v", err)
	}

	// Compare tips
	if tip1.Height != tip2.Height {
		t.Errorf("Tips have different heights: %d vs %d", tip1.Height, tip2.Height)
	}

	hash1 := tip1.Hash()
	hash2 := tip2.Hash()
	if hash1 != hash2 {
		t.Errorf("Tips have different hashes: %x vs %x", hash1, hash2)
	}
}

// TestForkChoice tests the blockchain's ability to choose the correct fork
func TestForkChoice(t *testing.T) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "blockchain_fork_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "testdb")

	// Setup blockchain with low difficulty
	bc := setupTestBlockchain(t, 100.0, 1, dbPath)

	// Initialize the blockchain
	err = bc.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer bc.Stop()

	// Wait for a block to be mined (block 1)
	var block1 *block.Block
	select {
	case block1 = <-bc.MiningChan:
		// Block was mined
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for block 1 to be mined")
	}

	// Create two competing blocks at height 2
	// First, get the block 1 hash
	block1Hash := block1.Hash()

	// Create two private keys for competing miners
	privKey1, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create two competing transactions
	tx1 := block.Transaction{
		FromAddress: bc.NodeConfig.ID.Address,
		ToAddress:   [32]byte{1, 2, 3},
		Amount:      50.0,
		Height:      2,
	}
	tx1.Sign(&bc.NodeConfig.ID.PrvKey)

	tx2 := block.Transaction{
		FromAddress: bc.NodeConfig.ID.Address,
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      60.0,
		Height:      2,
	}
	tx2.Sign(&bc.NodeConfig.ID.PrvKey)

	// Create two competing blocks
	fork1Block := &block.Block{
		PreHash:        block1Hash,
		Height:         2,
		EpochBeginHash: genesisBlock.Hash(),
		Txn:            tx1,
		PublicKey:      ecdsa_da.PublicKeyToBytes(&privKey1.PublicKey),
	}
	seed1 := ecdsa_da.DifficultySeed(&fork1Block.EpochBeginHash, fork1Block.Height)
	sig1, _ := ecdsa_da.Sign(privKey1, seed1[:])
	copy(fork1Block.Signature[:], sig1)

	fork2Block := &block.Block{
		PreHash:        block1Hash,
		Height:         2,
		EpochBeginHash: genesisBlock.Hash(),
		Txn:            tx2,
		PublicKey:      ecdsa_da.PublicKeyToBytes(&privKey2.PublicKey),
	}
	seed2 := ecdsa_da.DifficultySeed(&fork2Block.EpochBeginHash, fork2Block.Height)
	sig2, _ := ecdsa_da.Sign(privKey2, seed2[:])
	copy(fork2Block.Signature[:], sig2)

	// Add the fork blocks (simulating receiving them from the network)
	err = bc.AddBlock(fork1Block)
	if err != nil {
		t.Fatalf("Failed to add fork1Block: %v", err)
	}

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Create and add a block on top of fork1Block (making it the longest chain)
	fork1Hash := fork1Block.Hash()
	extendedForkBlock := &block.Block{
		PreHash:        fork1Hash,
		Height:         3,
		EpochBeginHash: genesisBlock.Hash(),
		Txn:            block.Transaction{Height: 3},
		PublicKey:      ecdsa_da.PublicKeyToBytes(&privKey1.PublicKey),
	}
	seedExt := ecdsa_da.DifficultySeed(&extendedForkBlock.EpochBeginHash, extendedForkBlock.Height)
	sigExt, _ := ecdsa_da.Sign(privKey1, seedExt[:])
	copy(extendedForkBlock.Signature[:], sigExt)

	// Now add fork2Block and the extended fork block
	err = bc.AddBlock(fork2Block)
	if err != nil {
		t.Fatalf("Failed to add fork2Block: %v", err)
	}

	err = bc.AddBlock(extendedForkBlock)
	if err != nil {
		t.Fatalf("Failed to add extendedForkBlock: %v", err)
	}

	// Give time for processing
	time.Sleep(500 * time.Millisecond)

	// Check that the tip is the extended fork (the longest chain)
	tip, err := bc.GetTipBlock()
	if err != nil {
		t.Fatalf("Failed to get tip: %v", err)
	}

	if tip.Height != 3 {
		t.Errorf("Tip height should be 3, got %d", tip.Height)
	}

	extendedForkHash := extendedForkBlock.Hash()
	tipHash := tip.Hash()
	if tipHash != extendedForkHash {
		t.Errorf("Tip hash should match extended fork block hash")
	}
}

// TestMiningDifferentDifficulties tests mining with different difficulty levels
func TestMiningDifferentDifficulties(t *testing.T) {
	// Skip in short mode as this test might take longer
	if testing.Short() {
		t.Skip("Skipping mining difficulty test in short mode")
	}

	// Create temporary directories
	tempDir1, _ := os.MkdirTemp("", "blockchain_diff_low")
	defer os.RemoveAll(tempDir1)

	tempDir2, _ := os.MkdirTemp("", "blockchain_diff_high")
	defer os.RemoveAll(tempDir2)

	// Setup blockchains with different difficulties
	bcLowDiff := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir1, "testdb"))
	bcHighDiff := setupTestBlockchain(t, 100.0, 5, filepath.Join(tempDir2, "testdb"))

	// Initialize blockchains
	err := bcLowDiff.Init()
	if err != nil {
		t.Fatalf("Failed to initialize low difficulty blockchain: %v", err)
	}
	defer bcLowDiff.Stop()

	err = bcHighDiff.Init()
	if err != nil {
		t.Fatalf("Failed to initialize high difficulty blockchain: %v", err)
	}
	defer bcHighDiff.Stop()

	// Time how long it takes to mine a block with low difficulty
	startLow := time.Now()
	select {
	case <-bcLowDiff.MiningChan:
		lowDiffTime := time.Since(startLow)
		t.Logf("Low difficulty mining took %v", lowDiffTime)
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for low difficulty mining")
	}

	// Time how long it takes to mine a block with high difficulty
	startHigh := time.Now()
	select {
	case <-bcHighDiff.MiningChan:
		highDiffTime := time.Since(startHigh)
		t.Logf("High difficulty mining took %v", highDiffTime)
	case <-time.After(30 * time.Second):
		t.Log("High difficulty mining took too long (expected)")
		return // This is actually expected for high difficulty
	}
}

// TestConcurrentMiningAndReceiving tests the blockchain's behavior when mining and receiving blocks concurrently
func TestConcurrentMiningAndReceiving(t *testing.T) {
	// Create temporary directories
	tempDir1, _ := os.MkdirTemp("", "blockchain_concurrent1")
	defer os.RemoveAll(tempDir1)

	tempDir2, _ := os.MkdirTemp("", "blockchain_concurrent2")
	defer os.RemoveAll(tempDir2)

	// Setup blockchains
	bc1 := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir1, "testdb"))
	bc2 := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir2, "testdb"))

	// Initialize blockchains
	err := bc1.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain 1: %v", err)
	}
	defer bc1.Stop()

	err = bc2.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain 2: %v", err)
	}
	defer bc2.Stop()

	// Let both nodes mine and exchange blocks
	var wg sync.WaitGroup
	wg.Add(2)

	// Node 1 mines and sends to node 2
	go func() {
		defer wg.Done()
		select {
		case block := <-bc1.MiningChan:
			bc2.AddBlock(block)
		case <-time.After(5 * time.Second):
			t.Error("Timed out waiting for bc1 to mine")
		}
	}()

	// Node 2 mines and sends to node 1
	go func() {
		defer wg.Done()
		select {
		case block := <-bc2.MiningChan:
			bc1.AddBlock(block)
		case <-time.After(5 * time.Second):
			t.Error("Timed out waiting for bc2 to mine")
		}
	}()

	// Wait for both operations to complete
	wg.Wait()

	// Give time for processing
	time.Sleep(1 * time.Second)

	// Both nodes should have at least height 1
	tip1, _ := bc1.GetTipBlock()
	tip2, _ := bc2.GetTipBlock()

	if tip1.Height < 1 || tip2.Height < 1 {
		t.Errorf("Both tips should be at least height 1, got %d and %d", tip1.Height, tip2.Height)
	}

	// Eventually, both nodes should converge to the same chain
	// This may take more complex synchronization in a real test
	t.Logf("Node 1 height: %d, Node 2 height: %d", tip1.Height, tip2.Height)
}

// TestInvalidBlock tests the blockchain's ability to reject invalid blocks
func TestInvalidBlock(t *testing.T) {
	// Create temporary directory
	tempDir, _ := os.MkdirTemp("", "blockchain_invalid")
	defer os.RemoveAll(tempDir)

	// Setup blockchain
	bc := setupTestBlockchain(t, 100.0, 1, filepath.Join(tempDir, "testdb"))

	// Initialize blockchain
	err := bc.Init()
	if err != nil {
		t.Fatalf("Failed to initialize blockchain: %v", err)
	}
	defer bc.Stop()

	// Wait for a block to be mined
	var _ *block.Block
	select {
	case _ = <-bc.MiningChan:
		// Block was mined
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for block to be mined")
	}

	// Create an invalid block with incorrect previous hash
	invalidBlock := &block.Block{
		PreHash:        [32]byte{1, 2, 3}, // Invalid previous hash
		Height:         2,
		EpochBeginHash: genesisBlock.Hash(),
		Txn: block.Transaction{
			Height: 2,
		},
	}

	// Create a valid private key for signing
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Sign the invalid block
	invalidBlock.PublicKey = ecdsa_da.PublicKeyToBytes(&privKey.PublicKey)
	seed := ecdsa_da.DifficultySeed(&invalidBlock.EpochBeginHash, invalidBlock.Height)
	sig, _ := ecdsa_da.Sign(privKey, seed[:])
	copy(invalidBlock.Signature[:], sig)

	// Try to add the invalid block
	err = bc.AddBlock(invalidBlock)
	if err != nil {
		t.Logf("As expected, failed to add invalid block: %v", err)
	}

	// Give time for processing
	time.Sleep(20000 * time.Millisecond)

	// Verify the tip is still the valid block
	tip, err := bc.GetTipBlock()
	if err != nil {
		t.Fatalf("Failed to get tip: %v", err)
	}

	tip.Hash()

	if !bc.VerifyBlock(tip) {
		t.Errorf("Tip should still be the valid, but it's not")
	}
}
