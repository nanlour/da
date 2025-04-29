package block

import (
    "bytes"
    "encoding/gob"
    "errors"
    "hash/crc32"
)

const (
	KB = 1024
	MB = 1024 * 1024 // Define 1 MB as 1024 * 1024 bytes
	MaxBlockBodySize = 10 * MB
)

type BlockHead struct {
    PreHash        [32]byte // Hash of the previous block head
    EpochBeginHash [32]byte // Hash marking the beginning of the epoch
	PublicKey      [64]byte // Public key associated with the block
	proof          [516]byte // Mining proof
	MerkleRoot     [32]byte // Root hash of the Merkle tree
}

type Transaction struct {
    FromAddress [32]byte  // Address of the sender
    ToAddress   [32]byte  // Address of the receiver
    Amount      float64 // Amount to be transferred
    Fee         float64 // Transaction fee
    Info        []byte  // Additional information about the transaction
}

// Body size must smaller then 10MB
type BlockBody struct {
    Transactions []Transaction  // A list of transactions included in the block
}

// Serialize serializes the BlockBody into a byte slice and appends a CRC32 checksum.
func (b *BlockBody) Serialize() ([]byte, error) {
    var buffer bytes.Buffer
    encoder := gob.NewEncoder(&buffer)

    // Encode the BlockBody
    if err := encoder.Encode(b); err != nil {
        return nil, err
    }

    // Calculate CRC32 checksum
    data := buffer.Bytes()
    checksum := crc32.ChecksumIEEE(data)

    // Append checksum to the serialized data
    result := append(data, byte(checksum>>24), byte(checksum>>16), byte(checksum>>8), byte(checksum))

    return result, nil
}

// Deserialize deserializes a byte slice into a BlockBody and verifies the CRC32 checksum.
func (b *BlockBody) Deserialize(data []byte) error {
    if len(data) < 4 {
        return errors.New("data too short to contain CRC")
    }

    // Separate the checksum from the data
    payload := data[:len(data)-4]
    checksum := uint32(data[len(data)-4])<<24 | uint32(data[len(data)-3])<<16 | uint32(data[len(data)-2])<<8 | uint32(data[len(data)-1])

    // Verify the checksum
    if crc32.ChecksumIEEE(payload) != checksum {
        return errors.New("CRC mismatch: data is corrupted")
    }

    // Decode the payload into the BlockBody
    buffer := bytes.NewBuffer(payload)
    decoder := gob.NewDecoder(buffer)
    if err := decoder.Decode(b); err != nil {
        return err
    }

    return nil
}