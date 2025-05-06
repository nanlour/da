package test

import (
    "bytes"
    "crypto/ecdsa"
    "crypto/sha256"
    "testing"

    "github.com/nanlour/da/block"
    ecdsaLib "github.com/nanlour/da/ecdsa"
    "github.com/nanlour/da/util"
)

// TestBlockCreation tests the creation and validation of blocks
func TestBlockCreation(t *testing.T) {
    // Initialize a test database in a temporary directory
    tempDBPath := t.TempDir() + "/testblockdb"
    err := util.InitialDB(tempDBPath)
    if err != nil {
        t.Fatalf("Failed to initialize test database: %v", err)
    }
    defer util.MainDB.Close()

    // Generate a keypair for testing
    privateKey, err := ecdsaLib.GenerateKeyPair()
    if err != nil {
        t.Fatalf("Failed to generate keypair: %v", err)
    }

    t.Run("Block Creation and Storage", func(t *testing.T) {
        testBlock, blockHash := createTestBlock(t, privateKey)

        // Test block storage
        err = util.MainDB.InsertHashBlock(blockHash[:], &testBlock)
        if err != nil {
            t.Fatalf("Failed to store block: %v", err)
        }

        // Test block retrieval
        retrievedBlock, err := util.MainDB.GetHashBlock(blockHash[:])
        if err != nil {
            t.Fatalf("Failed to retrieve block: %v", err)
        }

        // Verify block data integrity
        verifyBlockEquality(t, &testBlock, retrievedBlock)
    })
}

// TestBlockChain tests a simple blockchain with multiple blocks
func TestBlockChain(t *testing.T) {
    // Initialize a test database in a temporary directory
    tempDBPath := t.TempDir() + "/testchaindb"
    err := util.InitialDB(tempDBPath)
    if err != nil {
        t.Fatalf("Failed to initialize test database: %v", err)
    }
    defer util.MainDB.Close()

    // Generate a keypair for testing
    privateKey, err := ecdsaLib.GenerateKeyPair()
    if err != nil {
        t.Fatalf("Failed to generate keypair: %v", err)
    }

    t.Run("Create Blockchain", func(t *testing.T) {
        // Create a chain of 5 blocks
        blockHashes := createBlockchain(t, privateKey, 5)

        // Verify the chain tip
        verifyChainTip(t, blockHashes[4], 4)

        // Walk backward through the chain
        verifyChainWalkback(t, blockHashes)
    })

    t.Run("Block Transactions", func(t *testing.T) {
        // Create a single block with transaction
        testBlock, blockHash := createTestBlock(t, privateKey)
        
        // Store the block
        err = util.MainDB.InsertHashBlock(blockHash[:], &testBlock)
        if err != nil {
            t.Fatalf("Failed to store block: %v", err)
        }
        
        // Retrieve the block
        retrievedBlock, err := util.MainDB.GetHashBlock(blockHash[:])
        if err != nil {
            t.Fatalf("Failed to retrieve block: %v", err)
        }
        
        // Verify transaction data
        verifyTransaction(t, &testBlock.Txn, &retrievedBlock.Txn)
    })
}

// Helper functions

// createTestBlock creates a test block and returns it along with its hash
func createTestBlock(t *testing.T, privateKey *ecdsa.PrivateKey) (block.Block, [32]byte) {
    // Create a test block
    var testBlock block.Block

    // Set previous hash
    prevHash := sha256.Sum256([]byte("previous block data"))
    testBlock.PreHash = prevHash

    // Set epoch hash
    epochHash := sha256.Sum256([]byte("epoch data"))
    testBlock.EpochBeginHash = epochHash

    // Create a test transaction
    testBlock.Txn = block.Transaction{
        FromAddress: sha256.Sum256([]byte("sender")),
        ToAddress:   sha256.Sum256([]byte("receiver")),
        Amount:      100.0,
    }

    // Extract public key bytes for the block
    pubKeyBytes := privateKey.PublicKey.X.Bytes()
    pubKeyBytes = append(pubKeyBytes, privateKey.PublicKey.Y.Bytes()...)
    copy(testBlock.PublicKey[:], pubKeyBytes)

    // Create block data for signing
    blockData := append(prevHash[:], epochHash[:]...)
    
    // Sign the block data
    signature, err := ecdsaLib.Sign(privateKey, blockData)
    if err != nil {
        t.Fatalf("Failed to sign block data: %v", err)
    }

    // Set block signature
    copy(testBlock.Signature[:], signature)

    // Create test proof
    proof := [516]byte{}
    copy(proof[:], bytes.Repeat([]byte{1}, 516))
    testBlock.Proof = proof

    // Generate block hash
    blockHash := sha256.Sum256(blockData)
    
    return testBlock, blockHash
}

// createBlockchain creates a chain of blocks and returns their hashes
func createBlockchain(t *testing.T, privateKey *ecdsa.PrivateKey, length int) [][32]byte {
    var lastHash [32]byte
    blockHashes := make([][32]byte, length)

    for i := 0; i < length; i++ {
        var newBlock block.Block

        // Set previous hash
        newBlock.PreHash = lastHash

        // Set epoch hash (same for this test)
        epochHash := sha256.Sum256([]byte("test epoch"))
        newBlock.EpochBeginHash = epochHash

        // Extract public key bytes
        pubKeyBytes := privateKey.PublicKey.X.Bytes()
        pubKeyBytes = append(pubKeyBytes, privateKey.PublicKey.Y.Bytes()...)
        copy(newBlock.PublicKey[:], pubKeyBytes)

        // Create test transaction with unique amount
        newBlock.Txn = block.Transaction{
            FromAddress: sha256.Sum256([]byte("sender")),
            ToAddress:   sha256.Sum256([]byte("receiver")),
            Amount:      float64(100 * (i + 1)),
        }

        // Create block data for signing
        blockData := append(lastHash[:], epochHash[:]...)

        // Sign the block
        signature, err := ecdsaLib.Sign(privateKey, blockData)
        if err != nil {
            t.Fatalf("Failed to sign block %d: %v", i, err)
        }
        copy(newBlock.Signature[:], signature)

        // Create proof
        proof := [516]byte{}
        copy(proof[:], bytes.Repeat([]byte{byte(i)}, 516))
        newBlock.Proof = proof

        // Hash and store the block
        blockHash := sha256.Sum256(blockData)
        blockHashes[i] = blockHash
        
        err = util.MainDB.InsertHashBlock(blockHash[:], &newBlock)
        if err != nil {
            t.Fatalf("Failed to store block %d: %v", i, err)
        }

        // Set block metadata
        err = util.MainDB.InsertBlockHeight(blockHash[:], int64(i))
        if err != nil {
            t.Fatalf("Failed to set block %d height: %v", i, err)
        }

        err = util.MainDB.InsertHeightHash(int64(i), blockHash[:])
        if err != nil {
            t.Fatalf("Failed to store height->hash mapping for block %d: %v", i, err)
        }

        // Update tip if this is the last block
        if i == length-1 {
            err = util.MainDB.InsertTipHash(blockHash[:])
            if err != nil {
                t.Fatalf("Failed to update tip: %v", err)
            }
        }

        // Update lastHash for next iteration
        lastHash = blockHash
    }

    return blockHashes
}

// verifyBlockEquality checks if two blocks are equal
func verifyBlockEquality(t *testing.T, expected, actual *block.Block) {
    if !bytes.Equal(actual.PreHash[:], expected.PreHash[:]) {
        t.Errorf("PreHash mismatch: got %x, want %x", actual.PreHash, expected.PreHash)
    }

    if !bytes.Equal(actual.EpochBeginHash[:], expected.EpochBeginHash[:]) {
        t.Errorf("EpochBeginHash mismatch: got %x, want %x",
            actual.EpochBeginHash, expected.EpochBeginHash)
    }

    if !bytes.Equal(actual.PublicKey[:], expected.PublicKey[:]) {
        t.Errorf("PublicKey mismatch: got %x, want %x",
            actual.PublicKey, expected.PublicKey)
    }

    if !bytes.Equal(actual.Signature[:], expected.Signature[:]) {
        t.Errorf("Signature mismatch: got %x, want %x",
            actual.Signature, expected.Signature)
    }

    if !bytes.Equal(actual.Proof[:], expected.Proof[:]) {
        t.Errorf("Proof mismatch: got %x, want %x",
            actual.Proof, expected.Proof)
    }
    
    verifyTransaction(t, &expected.Txn, &actual.Txn)
}

// verifyTransaction checks if two transactions are equal
func verifyTransaction(t *testing.T, expected, actual *block.Transaction) {
    if !bytes.Equal(actual.FromAddress[:], expected.FromAddress[:]) {
        t.Errorf("Transaction FromAddress mismatch: got %x, want %x", 
            actual.FromAddress, expected.FromAddress)
    }
    
    if !bytes.Equal(actual.ToAddress[:], expected.ToAddress[:]) {
        t.Errorf("Transaction ToAddress mismatch: got %x, want %x", 
            actual.ToAddress, expected.ToAddress)
    }
    
    if actual.Amount != expected.Amount {
        t.Errorf("Transaction Amount mismatch: got %f, want %f", 
            actual.Amount, expected.Amount)
    }
}

// verifyChainTip verifies the tip of the blockchain
func verifyChainTip(t *testing.T, expectedTipHash [32]byte, expectedHeight int64) {
    tipHash, err := util.MainDB.GetTipHash()
    if err != nil {
        t.Fatalf("Failed to get tip hash: %v", err)
    }

    if !bytes.Equal(tipHash, expectedTipHash[:]) {
        t.Errorf("Tip hash mismatch: got %x, want %x", tipHash, expectedTipHash)
    }

    height, err := util.MainDB.GetBlockHeight(tipHash)
    if err != nil {
        t.Fatalf("Failed to get tip height: %v", err)
    }

    if height != expectedHeight {
        t.Errorf("Tip has incorrect height: got %d, want %d", height, expectedHeight)
    }
}

// verifyChainWalkback verifies the blockchain by walking backward from the tip
func verifyChainWalkback(t *testing.T, blockHashes [][32]byte) {
    tipHash, err := util.MainDB.GetTipHash()
    if err != nil {
        t.Fatalf("Failed to get tip hash: %v", err)
    }

    currentHash := tipHash
    length := len(blockHashes)

    for i := length - 1; i >= 0; i-- {
        block, err := util.MainDB.GetHashBlock(currentHash)
        if err != nil {
            t.Fatalf("Failed to get block at height %d: %v", i, err)
        }

        // Verify height matches
        height, err := util.MainDB.GetBlockHeight(currentHash)
        if err != nil {
            t.Fatalf("Failed to get height for block at index %d: %v", i, err)
        }

        if height != int64(i) {
            t.Errorf("Block height mismatch at index %d: got %d, want %d", i, height, i)
        }

        // Check hash matches expected hash at this index
        if !bytes.Equal(currentHash, blockHashes[i][:]) {
            t.Errorf("Block hash mismatch at index %d: got %x, want %x", 
                i, currentHash, blockHashes[i])
        }

        // For the next iteration, get previous block
        if i > 0 {
            currentHash = block.PreHash[:]
        }
    }
}