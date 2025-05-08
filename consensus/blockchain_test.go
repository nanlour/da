package consensus

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/ecdsa_da"
	"github.com/nanlour/da/util"
)

func TestMinerCreatesValidBlock(t *testing.T) {
	// Create a temporary DB for testing
	tempDbPath := t.TempDir() + "/testdb"
	err := util.InitialDB(tempDbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer util.MainDB.Close()

	// Setup genesis block in the database
	genesisHash := genesisBlock.Hash()
	if err := util.MainDB.InsertHashBlock(genesisHash[:], genesisBlock); err != nil {
		t.Fatalf("Failed to insert genesis block: %v", err)
	}
	if err := util.MainDB.InsertBlockHeight(genesisHash[:], 0); err != nil {
		t.Fatalf("Failed to set genesis block height: %v", err)
	}
	if err := util.MainDB.InsertHeightHash(0, genesisHash[:]); err != nil {
		t.Fatalf("Failed to set genesis height->hash mapping: %v", err)
	}
	if err := util.MainDB.InsertTipHash(genesisHash[:]); err != nil {
		t.Fatalf("Failed to set genesis as tip: %v", err)
	}

	// Create a private/public key pair for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Get address from public key
	address := ecdsa_da.PublicKeyToAddress(&privateKey.PublicKey)

	// Initialize blockchain with test configuration
	bc := &BlockChain{
		NodeConfig: &Config{
			ID: Account{
				PrvKey:  *privateKey,
				PubKey:  privateKey.PublicKey,
				Address: address,
			},
			StakeMine:        100.0,
			MiningDifficulty: 1, // Use lowest difficulty for fast testing
			InitStake: map[[32]byte]float64{
				address: 100.0,
			},
			StakeSum: 100.0,
		},
		TxnPool:    make(map[uint64]*block.Transaction),
		MiningChan: make(chan *block.Block, 10),
		P2PChan:    make(chan *block.Block, 10),
	}

	// Start mining in a goroutine
	go bc.mine()

	// Wait for a block to be mined with timeout
	var minedBlock *block.Block
	timeout := time.After(10 * time.Second)
	select {
	case minedBlock = <-bc.MiningChan:
		// Successfully received a mined block
	case <-timeout:
		t.Fatal("Timed out waiting for block to be mined")
	}

	// Verify the mined block
	if minedBlock == nil {
		t.Fatal("Mined block is nil")
	}

	// Check basic properties
	if minedBlock.Height != 1 {
		t.Errorf("Expected block height 1, got %d", minedBlock.Height)
	}

	if minedBlock.Txn.Height != minedBlock.Height {
		t.Errorf("Transaction height %d doesn't match block height %d",
			minedBlock.Txn.Height, minedBlock.Height)
	}

	// Verify the block passes our verification rules
	if !bc.VerifyBlock(minedBlock) {
		t.Error("Mined block failed verification")
	}

	// Check the signature
	pubKey, err := ecdsa_da.BytesToPublicKey(minedBlock.PublicKey)
	if err != nil {
		t.Fatalf("Failed to decode public key: %v", err)
	}

	seed := ecdsa_da.DifficultySeed(&minedBlock.EpochBeginHash, minedBlock.Height)
	if !ecdsa_da.Verify(pubKey, seed[:], minedBlock.Signature[:]) {
		t.Error("Block signature verification failed")
	}
}
