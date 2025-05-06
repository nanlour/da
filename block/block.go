package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

type Transaction struct {
	FromAddress [32]byte // Address of the sender
	ToAddress   [32]byte // Address of the receiver
	Amount      float64  // Amount to be transferred
}

type Block struct {
	PreHash        [32]byte // Hash of the previous block head
	Height         uint64
	EpochBeginHash [32]byte // Hash marking the beginning of the epoch
	Txn            Transaction
	Signature      [64]byte  // Signature of difficulty
	PublicKey      [64]byte  // Public key associated with the block
	Proof          [516]byte // Mining proof
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
	buf.Write(b.Txn.FromAddress[:])
	buf.Write(b.Txn.ToAddress[:])

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
