package rpc

import (
	"errors"

	"github.com/nanlour/da/src/block"
)

// BlockchainService defines the RPC methods for blockchain interaction
type BlockchainService struct {
	blockchain BlockchainInterface
}

type BlockchainInterface interface {
	GetBlockByHash(hash []byte) (*block.Block, error)
	GetTipBlock() (*block.Block, error)
	GetAddress() ([32]byte, error)
	GetAccountBalance(address *[32]byte) (float64, error)
	SendTxn(Txn *block.Transaction) error
}

func (s *BlockchainService) GetTip(args *struct{}, reply *[32]byte) error {
	TipBlock, err := s.blockchain.GetTipBlock()
	if err != nil {
		return err
	}
	var hashArray [32]byte
	hashArray = TipBlock.Hash()

	// Assign to the reply pointer
	*reply = hashArray

	return nil
}

func (s *BlockchainService) GetBlockByHash(hash [32]byte, reply *block.Block) error {
	// Get block head data from database
	blockHead, err := s.blockchain.GetBlockByHash(hash[:])
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
	balance, err := s.blockchain.GetAccountBalance(&address)
	if err != nil {
		return err
	}

	// Set the reply value
	*reply = balance

	return nil
}
