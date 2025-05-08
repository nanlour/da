package rpc

import (
	"errors"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/util"
)

// BlockInfo represents block data returned by RPC calls
type Tip struct {
	Hash   [32]byte
	Height uint64
}

// BlockchainService defines the RPC methods for blockchain interaction
type BlockchainService struct{}

func (s *BlockchainService) GetTip(args *struct{}, reply *Tip) error {
	tip, err := util.MainDB.GetTipHash()
	if err != nil {
		return err
	}

	height, err := util.MainDB.GetBlockHeight(tip)
	if err != nil {
		return err
	}

	// Convert slice to fixed-size array
	var hashArray [32]byte
	copy(hashArray[:], tip)

	// Assign to the reply pointer
	reply.Hash = hashArray
	reply.Height = uint64(height) // Convert int64 to uint64

	return nil
}

func (s *BlockchainService) GetBlockByHash(hash [32]byte, reply *block.Block) error {
	// Get block head data from database
	blockHead, err := util.MainDB.GetHashBlock(hash[:])
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
	balance, err := util.MainDB.GetAccountBalance(address[:])
	if err != nil {
		return err
	}

	// Set the reply value
	*reply = balance

	return nil
}
