package block

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

type Transaction struct {
	FromAddress [32]byte // Address of the sender
	ToAddress   [32]byte // Address of the receiver
	Amount      float64  // Amount to be transferred
	Height      uint64
	Signature   [64]byte
	PublicKey   [64]byte
}

// In theory i should add a signature for block content, ignore for prototype
type Block struct {
	PreHash        [32]byte // Hash of the previous block head
	Height         uint64
	EpochBeginHash [32]byte // Hash marking the beginning of the epoch
	Txn            Transaction
	Signature      [64]byte  // Signature of difficulty
	PublicKey      [64]byte  // Public key associated with the block
	Proof          [516]byte // Mining proof
}

// hash computes and returns the SHA-256 hash of the transaction data
func (txn *Transaction) hash() [32]byte {
	var buf bytes.Buffer

	// Add transaction fields to the buffer
	buf.Write(txn.FromAddress[:])
	buf.Write(txn.ToAddress[:])

	// Convert float64 amount to bytes
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, uint64(txn.Amount))
	buf.Write(amountBytes)

	// Convert uint64 Rand to bytes
	randBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(randBytes, txn.Height)
	buf.Write(randBytes)

	// Calculate the hash of the transaction data
	return sha256.Sum256(buf.Bytes())
}

// hash computes and returns the SHA-256 hash of the transaction data
func (txn *Transaction) Hash() [32]byte {
	var buf bytes.Buffer

	// Add transaction fields to the buffer
	buf.Write(txn.FromAddress[:])
	buf.Write(txn.ToAddress[:])

	// Convert float64 amount to bytes
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, uint64(txn.Amount))
	buf.Write(amountBytes)

	// Convert uint64 Rand to bytes
	randBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(randBytes, txn.Height)
	buf.Write(randBytes)

	buf.Write(txn.Signature[:])
	buf.Write(txn.PublicKey[:])

	// Calculate the hash of the transaction data
	return sha256.Sum256(buf.Bytes())
}

func (txn *Transaction) Sign(prvKey *ecdsa.PrivateKey) {
	// Calculate the hash of the transaction data
	txnHash := txn.hash()

	// Sign the hash with the private key
	r, s, err := ecdsa.Sign(rand.Reader, prvKey, txnHash[:])
	if err != nil {
		panic("Failed to sign transaction: " + err.Error())
	}

	// Convert signature (r, s) to bytes and store in transaction
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Clear the signature array
	for i := range txn.Signature {
		txn.Signature[i] = 0
	}

	// Copy R and S into the signature (right-aligned)
	copy(txn.Signature[32-len(rBytes):32], rBytes)
	copy(txn.Signature[64-len(sBytes):64], sBytes)

	// Store public key components
	pubKey := prvKey.PublicKey

	// Store X coordinate in first 32 bytes
	xBytes := pubKey.X.Bytes()
	copy(txn.PublicKey[32-len(xBytes):32], xBytes)

	// Store Y coordinate in last 32 bytes
	yBytes := pubKey.Y.Bytes()
	copy(txn.PublicKey[64-len(yBytes):64], yBytes)
}

// VerifySignature verifies if the transaction's signature is valid
func (txn *Transaction) Verify() bool {
	// Calculate the hash of the transaction data
	txnHash := txn.hash()

	// Extract public key components from the transaction
	pubKeyX := new(big.Int).SetBytes(txn.PublicKey[:32])
	pubKeyY := new(big.Int).SetBytes(txn.PublicKey[32:])

	// Reconstruct public key
	pubKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(), // Assuming P256 curve is used
		X:     pubKeyX,
		Y:     pubKeyY,
	}

	// Extract signature components
	r := new(big.Int).SetBytes(txn.Signature[:32])
	s := new(big.Int).SetBytes(txn.Signature[32:])

	// Verify the signature
	return ecdsa.Verify(pubKey, txnHash[:], r, s)
}

// Hash computes and returns the SHA-256 hash of the block
func (b *Block) Hash() [32]byte {
	var buf bytes.Buffer

	// Write all block fields to buffer in sequence
	buf.Write(b.PreHash[:])

	// Convert uint64 to bytes and write
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, b.Height)
	buf.Write(heightBytes)

	buf.Write(b.EpochBeginHash[:])

	// Write transaction data
	txnHash := b.Txn.Hash()
	buf.Write(txnHash[:])

	// Convert float64 to bytes
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, uint64(b.Txn.Amount))
	buf.Write(amountBytes)

	buf.Write(b.Signature[:])
	buf.Write(b.PublicKey[:])
	buf.Write(b.Proof[:])

	// Calculate SHA-256 hash
	return sha256.Sum256(buf.Bytes())
}

// Hash computes and returns the SHA-256 hash of the block
func (b *Block) HashwithoutProof() [32]byte {
	var buf bytes.Buffer

	// Write all block fields to buffer in sequence
	buf.Write(b.PreHash[:])

	// Convert uint64 to bytes and write
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, b.Height)
	buf.Write(heightBytes)

	buf.Write(b.EpochBeginHash[:])

	// Write transaction data
	txnHash := b.Txn.Hash()
	buf.Write(txnHash[:])

	// Convert float64 to bytes
	amountBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(amountBytes, uint64(b.Txn.Amount))
	buf.Write(amountBytes)

	buf.Write(b.Signature[:])
	buf.Write(b.PublicKey[:])

	// Calculate SHA-256 hash
	return sha256.Sum256(buf.Bytes())
}
