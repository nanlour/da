package consensus

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nanlour/da/block"
	"github.com/nanlour/da/db"
)

func (bc *BlockChain) TipManager() {
	log.Println("Starting blockchain tip manager...")

	for {
		select {
		case block := <-bc.MiningChan:
			// Process blocks from mining
			log.Printf("Received locally mined block at height %d\n", block.Height)
			if err := bc.processNewBlock(block, true); err != nil {
				log.Printf("Error processing mined block: %v\n", err)
			}

		case block := <-bc.P2PChan:
			// Process blocks from P2P network
			log.Printf("Received block from P2P network at height %d\n", block.Height)
			if err := bc.processNewBlock(block, false); err != nil {
				log.Printf("Error processing P2P block: %v\n", err)
			}
		case <-time.After(5 * time.Second):
			// Timeout case - useful for periodic health checks or preventing deadlocks
			log.Printf("TipManager heartbeat - no new blocks in the last 5 seconds, trying to fetch from peers")
			peers := bc.P2PNode.Peers()

			if len(peers) > 0 {
				// Random peer selection
				randomIndex := time.Now().UnixNano() % int64(len(peers))
				selectedPeer := peers[randomIndex]
				go bc.idealFetch(selectedPeer)
				log.Printf("Requesting tip from peer: %s", selectedPeer)

			} else {
				log.Printf("No peers available for tip synchronization")
			}
		}
	}
}

// processNewBlock handles a new block and resolves any forks
// isLocal indicates if the block was mined locally or received from network
func (bc *BlockChain) processNewBlock(newBlock *block.Block, isLocal bool) error {
	// Calculate block hash
	blockHash := newBlock.Hash()

	if !bc.VerifyBlock(newBlock) {
		log.Printf("Invalid Block %x\n", blockHash)
		return nil
	}

	// Get current tip
	tipHash, err := db.MainDB.GetTipHash()
	if err != nil {
		return fmt.Errorf("failed to get current tip: %w", err)
	}

	// Check if this block builds on our current tip
	if bytes.Equal(newBlock.PreHash[:], tipHash) {
		// This block extends our current main chain
		log.Printf("Block %x extends the main chain to height %d\n", blockHash, newBlock.Height)
		bc.DoTxn(&newBlock.Txn)

		err := db.MainDB.InsertHashBlock(&blockHash, newBlock)
		err = db.MainDB.InsertTipHash(&blockHash)

		bc.P2PNode.BroadcastBlock(newBlock)
		return err
	}

	// Potential fork detected - need to determine the longest chain
	log.Printf("Potential fork detected at height %d, resolving...\n", newBlock.Height)

	tipBlock, err := db.MainDB.GetHashBlock(tipHash)
	if err != nil {
		return fmt.Errorf("failed to get current tip: %w", err)
	}

	if newBlock.Height <= tipBlock.Height {
		log.Printf("Potential fork height too low, current Tip at %d\n", tipBlock.Height)
		return nil
	}

	return nil
}

// Request tip block from selected peer
func (bc *BlockChain) idealFetch(selectedPeer peer.ID) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a channel to receive the result
	resultCh := make(chan struct {
		block *block.Block
		err   error
	})

	tipBlock, err := bc.P2PNode.GetTip(selectedPeer)
	resultCh <- struct {
		block *block.Block
		err   error
	}{tipBlock, err}

	// Wait for either result or timeout
	select {
	case result := <-resultCh:
		if result.err != nil {
			log.Printf("Failed to get tip from peer %s: %v", selectedPeer, result.err)
			return
		}

		// Process the received tip block
		if result.block != nil {
			log.Printf("Received tip block at height %d from peer %s",
				result.block.Height, selectedPeer)

			// Process through the regular block handling channel
			bc.P2PChan <- result.block
		}
	case <-ctx.Done():
		log.Printf("Timeout waiting for tip from peer %s", selectedPeer)
	}
}
