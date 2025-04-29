package test

import (
	"bytes"
	"testing"

	"github.com/nanlour/da/block"
)

func TestBlockBody_SerializeAndDeserialize(t *testing.T) {
    // Create a sample BlockBody
    originalBlockBody := &block.BlockBody{
        Transactions: []block.Transaction{
            {
                FromAddress: [32]byte{1, 2, 3},
                ToAddress:   [32]byte{4, 5, 6},
                Amount:      100.0,
                Fee:         1.0,
                Info:        []byte("Test transaction 1"),
            },
            {
                FromAddress: [32]byte{7, 8, 9},
                ToAddress:   [32]byte{10, 11, 12},
                Amount:      200.0,
                Fee:         2.0,
                Info:        []byte("Test transaction 2"),
            },
        },
    }

    // Test Serialize
    serializedData, err := originalBlockBody.Serialize()
    if err != nil {
        t.Fatalf("Serialize failed: %v", err)
    }

    // Ensure serialized data is not empty
    if len(serializedData) == 0 {
        t.Fatalf("Serialized data is empty")
    }

    // Test Deserialize
    deserializedBlockBody := &block.BlockBody{}
    err = deserializedBlockBody.Deserialize(serializedData)
    if err != nil {
        t.Fatalf("Deserialize failed: %v", err)
    }

    // Verify that the deserialized BlockBody matches the original
    if len(deserializedBlockBody.Transactions) != len(originalBlockBody.Transactions) {
        t.Fatalf("Transaction count mismatch: expected %d, got %d",
            len(originalBlockBody.Transactions), len(deserializedBlockBody.Transactions))
    }

    for i, tx := range deserializedBlockBody.Transactions {
        originalTx := originalBlockBody.Transactions[i]
        if !bytes.Equal(tx.FromAddress[:], originalTx.FromAddress[:]) ||
            !bytes.Equal(tx.ToAddress[:], originalTx.ToAddress[:]) ||
            tx.Amount != originalTx.Amount ||
            tx.Fee != originalTx.Fee ||
            !bytes.Equal(tx.Info, originalTx.Info) {
            t.Errorf("Transaction mismatch at index %d", i)
        }
    }

    // Test CRC mismatch
    corruptedData := append(serializedData[:len(serializedData)-1], 0xFF) // Corrupt the last byte
    err = deserializedBlockBody.Deserialize(corruptedData)
    if err == nil || err.Error() != "CRC mismatch: data is corrupted" {
        t.Errorf("Expected CRC mismatch error, got: %v", err)
    }
}