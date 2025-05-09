package block

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"reflect"
	"testing"
)

func TestTransactionHash(t *testing.T) {
	// Create a test transaction
	txn := Transaction{
		FromAddress: [32]byte{1, 2, 3},
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      100.0,
		Height:      10,
	}

	// Hash the transaction
	hash1 := txn.hash()
	hash2 := txn.hash()

	// Verify that hashing is deterministic
	if hash1 != hash2 {
		t.Errorf("Transaction hashing is not deterministic: %v != %v", hash1, hash2)
	}

	// Modify the transaction and verify that the hash changes
	txn.Amount = 200.0
	hash3 := txn.hash()
	if hash1 == hash3 {
		t.Errorf("Transaction hash did not change after modifying the transaction")
	}
}

func TestTransactionSigningAndVerification(t *testing.T) {
	// Generate a private key for testing
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create a test transaction
	txn := Transaction{
		FromAddress: [32]byte{1, 2, 3},
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      100.0,
		Height:      10,
	}

	// Sign the transaction
	txn.Sign(privateKey)

	// Verify the signature
	if !txn.Verify() {
		t.Errorf("Transaction signature verification failed")
	}

	// Modify the transaction and verify that the signature is no longer valid
	txn.Amount = 200.0
	if txn.Verify() {
		t.Errorf("Transaction signature verification should fail after modifying the transaction")
	}
}

func TestBlockHash(t *testing.T) {
	// Generate a private key for the transaction
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Create a test transaction
	txn := Transaction{
		FromAddress: [32]byte{1, 2, 3},
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      100.0,
		Height:      10,
	}
	txn.Sign(privateKey)

	// Create a test block
	block := Block{
		PreHash:        [32]byte{7, 8, 9},
		Height:         20,
		EpochBeginHash: [32]byte{10, 11, 12},
		Txn:            txn,
		Signature:      [64]byte{},
		PublicKey:      [64]byte{},
		Proof:          [516]byte{},
	}

	// Hash the block
	hash1 := block.Hash()
	hash2 := block.Hash()

	// Verify that hashing is deterministic
	if !bytes.Equal(hash1[:], hash2[:]) {
		t.Errorf("Block hashing is not deterministic")
	}

	// Modify the block and verify that the hash changes
	block.Height = 30
	hash3 := block.Hash()
	if bytes.Equal(hash1[:], hash3[:]) {
		t.Errorf("Block hash did not change after modifying the block")
	}
}

func TestEmptyTransaction(t *testing.T) {
	// Create an empty transaction
	txn := Transaction{}

	// Hash the empty transaction
	hash := txn.hash()

	// Just verify it doesn't panic
	if hash == [32]byte{} {
		t.Logf("Empty transaction produces a non-zero hash")
	}
}

func TestDifferentPrivateKeys(t *testing.T) {
	// Generate two different private keys
	privateKey1, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKey2, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create a transaction
	txn := Transaction{
		FromAddress: [32]byte{1, 2, 3},
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      100.0,
		Height:      10,
	}

	// Sign with first key
	txn.Sign(privateKey1)

	// Verify it works
	if !txn.Verify() {
		t.Errorf("Transaction verification failed with correct key")
	}

	// Store the original signature and public key
	origSig := txn.Signature
	origPubKey := txn.PublicKey

	// Sign with second key
	txn.Sign(privateKey2)

	// Verify it works with new signature
	if !txn.Verify() {
		t.Errorf("Transaction verification failed with new key")
	}

	// Verify signatures are different
	if reflect.DeepEqual(origSig, txn.Signature) {
		t.Errorf("Signatures should be different with different private keys")
	}

	// Verify public keys are different
	if reflect.DeepEqual(origPubKey, txn.PublicKey) {
		t.Errorf("Public keys should be different with different private keys")
	}
}

func TestBlockWithDifferentTransactions(t *testing.T) {
	// Generate a private key
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	// Create two different transactions
	txn1 := Transaction{
		FromAddress: [32]byte{1, 2, 3},
		ToAddress:   [32]byte{4, 5, 6},
		Amount:      100.0,
		Height:      10,
	}
	txn1.Sign(privateKey)

	txn2 := Transaction{
		FromAddress: [32]byte{7, 8, 9},
		ToAddress:   [32]byte{10, 11, 12},
		Amount:      200.0,
		Height:      20,
	}
	txn2.Sign(privateKey)

	// Create blocks with different transactions
	block1 := Block{
		PreHash:        [32]byte{13, 14, 15},
		Height:         30,
		EpochBeginHash: [32]byte{16, 17, 18},
		Txn:            txn1,
	}

	block2 := Block{
		PreHash:        [32]byte{13, 14, 15},
		Height:         30,
		EpochBeginHash: [32]byte{16, 17, 18},
		Txn:            txn2,
	}

	// Verify that blocks with different transactions have different hashes
	hash1 := block1.Hash()
	hash2 := block2.Hash()

	if bytes.Equal(hash1[:], hash2[:]) {
		t.Errorf("Blocks with different transactions should have different hashes")
	}
}
