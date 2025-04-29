package util

import (
	"os"
    "path/filepath"

	"github.com/nanlour/da/block"
)

// WriteBlock writes a BlockBody to the database using the hash as the key
func WriteBlock(hash [32]byte, body *block.BlockBody) error {
	// Create the directory if it doesn't exist
	// TODO: move it to node initial
	if err := os.MkdirAll(BlockDir, os.ModePerm); err != nil {
		return err
	}

	// Write to the file
	filename := filepath.Join(BlockDir, string(hash[:]))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

    data, err := body.Serialize()
    if err != nil {
        return err
    }
    _, err = file.Write(data)

	return err
}

// GetBlockFromFile retrieves a BlockBody from a file using the hash as the key
func GetBlockFromFile(hash [32]byte) (*block.BlockBody, error) {
    // Construct the file path
    filename := filepath.Join(BlockDir, string(hash[:]))

    // Open the file
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    // Read the file content into a byte slice
    fileInfo, err := file.Stat()
    if err != nil {
        return nil, err
    }

    data := make([]byte, fileInfo.Size())
    _, err = file.Read(data)
    if err != nil {
        return nil, err
    }

    // Deserialize the data into a BlockBody
    var body block.BlockBody
    if err := body.Deserialize(data); err != nil {
        return nil, err
    }

    return &body, nil
}