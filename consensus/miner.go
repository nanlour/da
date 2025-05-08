package consensus

import (
	"bytes"
	"context"
	"log"
	"time"

	"github.com/nanlour/da/block"
	"github.com/nanlour/da/ecdsa_da"
	"github.com/nanlour/da/util"
	"github.com/nanlour/da/vdf_go"
)

func (bc *BlockChain) mine() {
	log.Println("Starting mining process...")

	// Run the mining loop indefinitely
	for {
		// Get the current tip hash and block
		tipHash, err := util.MainDB.GetTipHash()
		if err != nil {
			log.Printf("Failed to get tip hash: %v, retrying in 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}

		tipBlock, err := util.MainDB.GetHashBlock(tipHash)
		if err != nil {
			log.Printf("Failed to get tip block: %v, retrying in 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Create a new block to mine
		newBlock := &block.Block{
			PreHash:        bytesToHash32(tipHash),
			Height:         tipBlock.Height + 1,
			EpochBeginHash: genesisBlock.Hash(), // Use genesisBlock for now
			Txn:            bc.selectTransaction(tipBlock.Height + 1),
			PublicKey:      ecdsa_da.PublicKeyToBytes(&bc.NodeConfig.ID.PubKey),
		}

		// Sign the difficulty using the node's private key
		seed := ecdsa_da.DifficultySeed(&newBlock.EpochBeginHash, newBlock.Height)
		signature, err := ecdsa_da.Sign(&bc.NodeConfig.ID.PrvKey, seed[:])
		if err != nil {
			log.Printf("Failed to sign block: %v", err)
			continue
		}
		copy(newBlock.Signature[:], signature)
		difficulty := ecdsa_da.Difficulty(signature, bc.NodeConfig.StakeSum, bc.NodeConfig.StakeMine, bc.NodeConfig.MiningDifficulty)

		// Create context for VDF that can be cancelled
		ctx, cancel := context.WithCancel(context.Background())
		stopChan := make(chan struct{})

		// Set up goroutine to monitor for tip changes
		go func(currentTipHash []byte, stopMining func()) {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					latestTipHash, err := util.MainDB.GetTipHash()
					if err != nil {
						log.Printf("Error checking tip hash: %v", err)
						continue
					}

					// If tip has changed, stop mining
					if !bytes.Equal(currentTipHash, latestTipHash) {
						log.Println("Tip has changed, stopping current mining operation")
						stopMining()
						return
					}
				}
			}
		}(tipHash, func() {
			close(stopChan)
			cancel()
		})

		// Create VDF with mining difficulty
		vdf := vdf_go.New(int(difficulty), newBlock.HashwithoutProof())

		log.Printf("Mining block at height %d with difficulty %d",
			newBlock.Height, difficulty)

		// Start VDF computation in a separate goroutine
		go vdf.Execute(stopChan)

		// Wait for VDF completion or cancellation
		select {
		case proof := <-vdf.GetOutputChannel():
			// Mining completed, copy proof to block
			copy(newBlock.Proof[:], proof[:])

			log.Printf("Successfully mined block at height %d", newBlock.Height)

			// Send the mined block to the channel
			bc.MiningChan <- newBlock

		case <-ctx.Done():
			// Mining was cancelled, clean up
			log.Println("Mining operation cancelled")
		}

		// Cancel context if not already done
		cancel()

		// Short delay before starting next mining cycle
		time.Sleep(10 * time.Millisecond)
	}
}

// Helper function to convert byte slice to [32]byte
func bytesToHash32(data []byte) [32]byte {
	var result [32]byte
	copy(result[:], data)
	return result
}

// Select a transaction from the transaction pool
func (bc *BlockChain) selectTransaction(height uint64) block.Transaction {
	// Try to find a transaction for this height in the pool
	if txn, exists := bc.TxnPool[height]; exists && txn != nil {
		return *txn
	}

	// No transaction found for this height, create an empty one
	emptyTxn := block.Transaction{
		FromAddress: [32]byte{},
		ToAddress:   [32]byte{},
		Amount:      0,
		Height:      height,
	}

	emptyTxn.Sign(&bc.NodeConfig.ID.PrvKey)
	return emptyTxn
}
