package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/nanlour/da/src/block"
	"github.com/nanlour/da/src/db"
	"github.com/nanlour/da/src/ecdsa_da"
	"github.com/nanlour/da/src/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestBlockchain creates a minimal blockchain for testing with just the DB component
func setupTestBlockchain(t *testing.T) (*BlockChain, func()) {
	// Create temp directory for DB
	tempDir, err := os.MkdirTemp("", "blockchain_test_")
	require.NoError(t, err)

	// Generate a private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	address := ecdsa_da.PublicKeyToAddress(&privateKey.PublicKey)

	// Create blockchain config
	config := &Config{
		ID: Account{
			PrvKey:  *privateKey,
			PubKey:  privateKey.PublicKey,
			Address: address,
		},
		StakeMine:        1.0,
		MiningDifficulty: 10,
		DbPath:           filepath.Join(tempDir, "testdb"),
		InitStake: map[[32]byte]float64{
			address: 100.0,
		},
		StakeSum: 100.0,
		InitBank: map[[32]byte]float64{
			address: 1000.0,
		},
	}

	// Initialize blockchain and database
	bc := &BlockChain{}
	bc.SetConfig(config)

	// Set up database
	dbManager, err := db.InitialDB(config.DbPath)
	require.NoError(t, err)
	bc.mainDB = dbManager

	// Initialize transaction pool
	bc.TxnPool = TransactionPool{
		txnMap: make(map[uint64]*block.Transaction),
	}

	// Initialize channels
	bc.P2PChan = make(chan *p2p.P2PBlock, 10)
	bc.MiningChan = make(chan *block.Block, 10)

	// Set up genesis block
	gBHash := genesisBlock.Hash()
	err = bc.mainDB.InsertTipHash(&gBHash)
	require.NoError(t, err)
	err = bc.mainDB.InsertHashBlock(&gBHash, &genesisBlock)
	require.NoError(t, err)

	// Set up initial balances
	for addr, balance := range config.InitBank {
		err = bc.mainDB.InsertAccountBalance(&addr, balance)
		require.NoError(t, err)
	}

	// Return cleanup function
	cleanup := func() {
		bc.mainDB.Close()
		os.RemoveAll(tempDir)
	}

	return bc, cleanup
}

// TestBlockchainDBIntegration tests the integration between blockchain and database
func TestBlockchainDBIntegration(t *testing.T) {
	bc, cleanup := setupTestBlockchain(t)
	defer cleanup()

	// Test blockchain initialization
	tipBlock, err := bc.GetTipBlock()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), tipBlock.Height)

	// Test GetAddress
	address, err := bc.GetAddress()
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, address)

	// Test GetAccountBalance
	balance, err := bc.GetAccountBalance(&address)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, balance)

	// Test transaction handling
	testTransaction(t, bc)
}

// testTransaction tests transaction functionality
func testTransaction(t *testing.T, bc *BlockChain) {
	// Get our address
	fromAddress, err := bc.GetAddress()
	require.NoError(t, err)

	// Create a recipient address
	var toAddress [32]byte
	copy(toAddress[:], []byte("recipient-address-12345678901234567"))

	// Initialize recipient account
	err = bc.mainDB.InsertAccountBalance(&toAddress, 0)
	require.NoError(t, err)

	// Create a transaction
	tx := &block.Transaction{
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Amount:      100.0,
		Height:      1,
	}

	// Sign the transaction
	tx.Sign(&bc.NodeConfig.ID.PrvKey)

	// Add transaction to the pool
	err = bc.AddTxn(tx)
	require.NoError(t, err)

	// Verify transaction is in the pool
	pooledTx, exists := bc.TxnPool.GetTransaction(1)
	assert.True(t, exists)
	assert.Equal(t, tx.Amount, pooledTx.Amount)

	// Process the transaction
	err = bc.DoTxn(tx)
	require.NoError(t, err)

	// Verify balances after transaction
	fromBalance, err := bc.GetAccountBalance(&fromAddress)
	require.NoError(t, err)
	assert.Equal(t, 900.0, fromBalance) // 1000 - 100

	toBalance, err := bc.GetAccountBalance(&toAddress)
	require.NoError(t, err)
	assert.Equal(t, 100.0, toBalance) // 0 + 100

	// Test transaction rollback
	err = bc.UNDoTxn(tx)
	require.NoError(t, err)

	// Verify balances after rollback
	fromBalance, err = bc.GetAccountBalance(&fromAddress)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, fromBalance) // 900 + 100 (restored)

	toBalance, err = bc.GetAccountBalance(&toAddress)
	require.NoError(t, err)
	assert.Equal(t, 0.0, toBalance) // 100 - 100 (restored)
}

// TestMultipleTransactions tests processing multiple transactions
func TestMultipleTransactions(t *testing.T) {
	bc, cleanup := setupTestBlockchain(t)
	defer cleanup()

	// Get our address
	fromAddress, err := bc.GetAddress()
	require.NoError(t, err)

	// Create multiple recipient addresses
	recipients := make([][32]byte, 3)
	for i := range recipients {
		copy(recipients[i][:], []byte("recipient-"+string(rune('A'+i))+"-12345678901234567"))
		err = bc.mainDB.InsertAccountBalance(&recipients[i], 0)
		require.NoError(t, err)
	}

	// Create and process multiple transactions
	amounts := []float64{100.0, 200.0, 300.0}
	for i, amount := range amounts {
		tx := &block.Transaction{
			FromAddress: fromAddress,
			ToAddress:   recipients[i],
			Amount:      amount,
			Height:      uint64(i + 1),
		}
		tx.Sign(&bc.NodeConfig.ID.PrvKey)

		// Add to pool and process
		bc.AddTxn(tx)
		bc.DoTxn(tx)
	}

	// Verify sender balance
	fromBalance, err := bc.GetAccountBalance(&fromAddress)
	require.NoError(t, err)
	assert.Equal(t, 400.0, fromBalance) // 1000 - (100+200+300)

	// Verify recipient balances
	for i, amount := range amounts {
		balance, err := bc.GetAccountBalance(&recipients[i])
		require.NoError(t, err)
		assert.Equal(t, amount, balance)
	}
}

// TestBlockRetrieval tests block storage and retrieval
func TestBlockRetrieval(t *testing.T) {
	bc, cleanup := setupTestBlockchain(t)
	defer cleanup()

	// Get genesis block
	genesisBlock, err := bc.GetTipBlock()
	require.NoError(t, err)
	genesisHash := genesisBlock.Hash()

	// Test retrieving block by hash
	retrievedBlock, err := bc.GetBlockByHash(genesisHash[:])
	require.NoError(t, err)
	assert.Equal(t, genesisBlock.Height, retrievedBlock.Height)

	// Test non-existent block
	var nonExistentHash [32]byte
	rand.Read(nonExistentHash[:])
	nonExistentBlock, err := bc.GetBlockByHash(nonExistentHash[:])
	if err != nil {
		// Some implementations might return an error
		assert.Error(t, err)
	} else {
		// Others might return nil without error
		assert.Nil(t, nonExistentBlock, "Non-existent block should be nil")
	}
}
