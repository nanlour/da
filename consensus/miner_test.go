package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/db"
	"github.com/nanlour/da/ecdsa_da"
)

func TestMiningChan(t *testing.T) {
	// Create a temporary DB for testing
	tempDbPath := t.TempDir() + "/testdb"
	err := db.InitialDB(tempDbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer db.MainDB.Close()

	// Setup genesis block
	genesisHash := genesisBlock.Hash()
	if err := db.MainDB.InsertHashBlock(genesisHash[:], genesisBlock); err != nil {
		t.Fatalf("Failed to insert genesis block: %v", err)
	}
	if err := db.MainDB.InsertTipHash(genesisHash[:]); err != nil {
		t.Fatalf("Failed to set genesis as tip: %v", err)
	}

	// Create a private/public key pair for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Create test transaction for the transaction pool
	testTxn := &block.Transaction{
		FromAddress: [32]byte{1, 2, 3}, // Example sender
		ToAddress:   [32]byte{4, 5, 6}, // Example receiver
		Amount:      100.0,             // Example amount
		Height:      1,                 // Should match the next block height
	}
	testTxn.Sign(privateKey)

	// Initialize blockchain with test configuration
	bc := &BlockChain{
		NodeConfig: &Config{
			ID: Account{
				PrvKey: *privateKey,
				PubKey: privateKey.PublicKey,
			},
			StakeMine:        100.0,
			MiningDifficulty: 1, // Low difficulty for quick tests
			InitStake: map[[32]byte]float64{
				ecdsa_da.PublicKeyToAddress(&privateKey.PublicKey): 100.0,
			},
			StakeSum: 100.0,
		},
		TxnPool: TransactionPool{
			txnMap: make(map[uint64]*block.Transaction),
		},
		MiningChan: make(chan *block.Block, 10),
		P2PChan:    make(chan *block.Block, 10),
	}

	// Test Case 1: Mining with no transaction in pool
	t.Run("Mining with no transaction in pool", func(t *testing.T) {
		// Start mining in a goroutine
		go func() {
			// Call selectTransaction directly to test
			emptyTxn := bc.selectTransaction(1)

			// Manually create a block using this transaction
			tipHash, _ := db.MainDB.GetTipHash()
			tipBlock, _ := db.MainDB.GetHashBlock(tipHash)

			newBlock := &block.Block{
				PreHash:        bytesToHash32(tipHash),
				Height:         tipBlock.Height + 1,
				EpochBeginHash: genesisBlock.EpochBeginHash,
				Txn:            emptyTxn,
				PublicKey:      ecdsa_da.PublicKeyToBytes(&bc.NodeConfig.ID.PubKey),
			}

			// Put the block in the mining channel as if it was mined
			bc.MiningChan <- newBlock
		}()

		// Wait for the block to be mined
		var minedBlock *block.Block
		select {
		case minedBlock = <-bc.MiningChan:
			// Block received
		case <-time.After(2 * time.Second):
			t.Fatal("Timed out waiting for mined block")
		}

		// Verify the created block
		if minedBlock.Height != 1 {
			t.Errorf("Expected block height 1, got %d", minedBlock.Height)
		}

		// Verify the transaction is an empty one as expected
		if minedBlock.Txn.Amount != 0 {
			t.Errorf("Expected empty transaction with amount 0, got %f", minedBlock.Txn.Amount)
		}

		// Verify the transaction signature
		if !minedBlock.Txn.Verify() {
			t.Error("Generated empty transaction failed signature verification")
		}
	})

	// Test Case 2: Mining with a transaction in pool
	t.Run("Mining with transaction in pool", func(t *testing.T) {
		// Add transaction to the pool
		bc.TxnPool.AddTransaction(1, testTxn)

		// Start mining in a goroutine
		go func() {
			// Call selectTransaction directly to test with a transaction in the pool
			txn := bc.selectTransaction(1)

			// Manually create a block using this transaction
			tipHash, _ := db.MainDB.GetTipHash()
			tipBlock, _ := db.MainDB.GetHashBlock(tipHash)

			newBlock := &block.Block{
				PreHash:        bytesToHash32(tipHash),
				Height:         tipBlock.Height + 1,
				EpochBeginHash: genesisBlock.EpochBeginHash,
				Txn:            txn,
				PublicKey:      ecdsa_da.PublicKeyToBytes(&bc.NodeConfig.ID.PubKey),
			}

			// Put the block in the mining channel as if it was mined
			bc.MiningChan <- newBlock
		}()

		// Wait for the block to be mined
		var minedBlock *block.Block
		select {
		case minedBlock = <-bc.MiningChan:
			// Block received
		case <-time.After(2 * time.Second):
			t.Fatal("Timed out waiting for mined block")
		}

		// Verify the created block
		if minedBlock.Height != 1 {
			t.Errorf("Expected block height 1, got %d", minedBlock.Height)
		}

		// Verify that the transaction from pool was used
		if minedBlock.Txn.Amount != 100.0 {
			t.Errorf("Expected transaction with amount 100.0, got %f", minedBlock.Txn.Amount)
		}

		// Verify the transaction signature
		if !minedBlock.Txn.Verify() {
			t.Error("Transaction from pool failed signature verification")
		}
	})
}
