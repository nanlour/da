package consensus

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nanlour/da/block"
	"github.com/nanlour/da/util"
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

	// Get current tip
	tipHash, err := util.MainDB.GetTipHash()
	if err != nil {
		return fmt.Errorf("failed to get current tip: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to get tip block: %w", err)
	}

	// Store the block in the database regardless of whether it's on the main chain
	if err := util.MainDB.InsertHashBlock(blockHash[:], newBlock); err != nil {
		return fmt.Errorf("failed to store block: %w", err)
	}

	// Set block height mapping
	if err := util.MainDB.InsertBlockHeight(blockHash[:], int64(newBlock.Height)); err != nil {
		return fmt.Errorf("failed to set block height: %w", err)
	}

	if err := util.MainDB.InsertHeightHash(int64(newBlock.Height), blockHash[:]); err != nil {
		return fmt.Errorf("failed to set height->hash mapping: %w", err)
	}

	// Check if this block builds on our current tip
	if bytes.Equal(newBlock.PreHash[:], tipHash) {
		// This block extends our current main chain
		log.Printf("Block %x extends the main chain to height %d\n", blockHash, newBlock.Height)
		return util.MainDB.InsertTipHash(blockHash[:])
	}

	// Potential fork detected - need to determine the longest chain
	log.Printf("Potential fork detected at height %d, resolving...\n", newBlock.Height)

	// Get height of current tip
	tipHeight, err := util.MainDB.GetBlockHeight(tipHash)
	if err != nil {
		return fmt.Errorf("failed to get tip height: %w", err)
	}

	// If the new block is on a fork with greater height, it becomes the new tip
	if newBlock.Height > uint64(tipHeight) {
		log.Printf("Fork resolution: Block %x creates a longer chain (height %d > %d)\n",
			blockHash, newBlock.Height, tipHeight)
		return util.MainDB.InsertTipHash(blockHash[:])
	} else if newBlock.Height == uint64(tipHeight) && isLocal {
		// If heights are equal and this is our own mined block, prefer it
		log.Printf("Fork resolution: Equal height chains, preferring locally mined block %x\n", blockHash)
		return util.MainDB.InsertTipHash(blockHash[:])
	}

	log.Printf("Fork resolution: Keeping existing tip (height %d >= %d)\n", tipHeight, newBlock.Height)
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
