package rpc

import (
	"errors"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/db"
)

// BlockchainService defines the RPC methods for blockchain interaction
type BlockchainService struct{}

func (s *BlockchainService) GetTip(args *struct{}, reply *[32]byte) error {
	tip, err := db.MainDB.GetTipHash()
	if err != nil {
		return err
	}
	var hashArray [32]byte
	copy(hashArray[:], tip)

	// Assign to the reply pointer
	*reply = hashArray

	return nil
}

func (s *BlockchainService) GetBlockByHash(hash [32]byte, reply *block.Block) error {
	// Get block head data from database
	blockHead, err := db.MainDB.GetHashBlock(hash[:])
	if err != nil {
		return err
	}

	// If block doesn't exist
	if blockHead == nil {
		return errors.New("block not found")
	}

	// Copy the block head data to the reply pointer
	*reply = *blockHead

	return nil
}

func (s *BlockchainService) GetBalanceByAddress(address [32]byte, reply *float64) error {
	// Get balance from database
	balance, err := db.MainDB.GetAccountBalance(&address)
	if err != nil {
		return err
	}

	// Set the reply value
	*reply = balance

	return nil
}
