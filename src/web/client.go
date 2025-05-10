package web

import (
	"errors"
	"net/rpc"

	"github.com/nanlour/da/src/block"
)

// RPCClient handles communication with the blockchain RPC server
type RPCClient struct {
	client *rpc.Client
}

// NewRPCClient creates a new client connected to the RPC server
func NewRPCClient(address string) (*RPCClient, error) {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &RPCClient{client: client}, nil
}

// GetTip returns the hash of the latest block
func (c *RPCClient) GetTip() ([32]byte, error) {
	var result [32]byte
	err := c.client.Call("BlockchainService.GetTip", struct{}{}, &result)
	return result, err
}

// GetBlockByHash returns a block by its hash
func (c *RPCClient) GetBlockByHash(hash [32]byte) (*block.Block, error) {
	var result block.Block
	err := c.client.Call("BlockchainService.GetBlockByHash", hash, &result)
	return &result, err
}

// GetBalanceByAddress returns the balance for a given address
func (c *RPCClient) GetBalanceByAddress(address [32]byte) (float64, error) {
	var result float64
	err := c.client.Call("BlockchainService.GetBalanceByAddress", address, &result)
	return result, err
}

// SendTxn sends a transaction to the specified address with the given amount
func (c *RPCClient) SendTxn(destination [32]byte, amount float64) (bool, error) {
	args := struct {
		Destination [32]byte
		Amount      float64
	}{
		Destination: destination,
		Amount:      amount,
	}
	var result bool
	err := c.client.Call("BlockchainService.SendTxn", args, &result)
	return result, err
}

// GetAddress returns the current node's address
func (c *RPCClient) GetAddress() ([32]byte, error) {
	var result [32]byte
	// Call the blockchain's GetAddress method
	err := c.client.Call("BlockchainService.GetAddress", struct{}{}, &result)
	return result, err
}

// GetLastTenBlocks returns the most recent 10 blocks
func (c *RPCClient) GetLastTenBlocks() ([]*block.Block, error) {
	// First get the tip block
	tipHash, err := c.GetTip()
	if err != nil {
		return nil, err
	}

	blocks := make([]*block.Block, 0, 10)
	currentHash := tipHash

	// Get the last 10 blocks by following the chain backwards
	for i := 0; i < 10; i++ {
		currentBlock, err := c.GetBlockByHash(currentHash)
		if err != nil {
			if i == 0 {
				return nil, errors.New("failed to get tip block")
			}
			// If we can't get more blocks but have some, return what we have
			break
		}

		blocks = append(blocks, currentBlock)

		// If this is the genesis block (PreHash is zero), stop
		var zeroHash [32]byte
		if currentBlock.PreHash == zeroHash {
			break
		}

		// Move to the previous block
		currentHash = currentBlock.PreHash
	}

	return blocks, nil
}

// Close closes the RPC connection
func (c *RPCClient) Close() error {
	return c.client.Close()
}
