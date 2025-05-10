package consensus

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nanlour/da/src/block"
	"github.com/nanlour/da/src/p2p"
)

func (bc *BlockChain) TipManager() {
	log.Println("Starting blockchain tip manager...")

	for {
		select {
		case block := <-bc.MiningChan:
			// Process blocks from mining
			log.Printf("Received locally mined block at height %d\n", block.Height)
			if err := bc.processNewBlock(block, true, ""); err != nil {
				log.Printf("Error processing mined block: %v\n", err)
			}

		case p2pblock := <-bc.P2PChan:
			// Process blocks from P2P network
			log.Printf("Received block from P2P network at height %d\n", p2pblock.Block.Height)
			if err := bc.processNewBlock(&p2pblock.Block, false, p2pblock.Sender); err != nil {
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
func (bc *BlockChain) processNewBlock(newBlock *block.Block, isLocal bool, sender string) error {
	// Get current tip
	tipHash := bc.MyChain[len(bc.MyChain)-1].Hash

	tipBlock, err := bc.mainDB.GetHashBlock(tipHash[:])
	if err != nil {
		return fmt.Errorf("failed to get current tip: %w", err)
	}

	// Calculate block hash
	blockHash := newBlock.Hash()

	if newBlock.Height <= tipBlock.Height {
		log.Printf("Potential fork height too low, current Tip at %d\n", tipBlock.Height)
		return nil
	}

	if !bc.VerifyBlock(newBlock) {
		log.Printf("Invalid Block %x\n", blockHash)
		return nil
	}

	// Check if this block builds on our current tip
	if bytes.Equal(newBlock.PreHash[:], tipHash[:]) {
		// This block extends our current main chain
		log.Printf("Block %x extends the main chain to height %d\n", blockHash, newBlock.Height)
		bc.DoTxn(&newBlock.Txn)

		err := bc.mainDB.InsertHashBlock(&blockHash, newBlock)
		err = bc.mainDB.InsertTipHash(&blockHash)

		bc.P2PNode.BroadcastBlock(newBlock)
		bc.MyChain = append(bc.MyChain, &Chain{Hash: blockHash, PrvHash: newBlock.PreHash})
		return err
	} else if isLocal { // Ignore self mined block
		return nil
	}

	// Potential fork detected - need to determine the longest chain
	log.Printf("Potential fork detected at height %d, resolving...\n", newBlock.Height)

	bc.checkFork(newBlock, sender)

	return nil
}

func (bc *BlockChain) checkFork(newBlock *block.Block, sender string) {
	blockHash := newBlock.Hash()
	log.Printf("Starting fork resolution for block %x at height %d from sender %s",
		blockHash, newBlock.Height, sender)

	newchain := map[uint64]*block.Block{
		newBlock.Height: newBlock,
	}
	height := newBlock.Height

	for {
		log.Printf("Fetching previous block at height %d with hash %x", height-1, newchain[height].PreHash)
		peerID, err := peer.Decode(sender)
		if err != nil {
			log.Printf("Fail to restore peerid")
		}
		block, err := bc.P2PNode.GetBlockByHash(newchain[height].PreHash, peerID)
		if err != nil {
			log.Printf("Failed to get block at height %d: %v", height-1, err)
			return
		}

		height -= 1
		if block.Height != height {
			log.Printf("Block height mismatch: expected %d, got %d", height, block.Height)
			return
		}

		if !bc.VerifyBlock(block) {
			log.Printf("Block verification failed when check fork at height %d", height)
			return
		}

		log.Printf("Adding block %x at height %d to potential new chain", block.Hash(), height)
		newchain[height] = block

		if len(bc.MyChain) >= int(height) && bytes.Equal(block.PreHash[:], bc.MyChain[height-1].Hash[:]) { // Find it in our chain
			log.Printf("Found fork point at height %d - reorganizing chain", height)

			// Rollback transactions from our current chain
			log.Printf("Rolling back transactions from height %d to %d", height, len(bc.MyChain)-1)
			for i := height; i < uint64(len(bc.MyChain)); i++ {
				oldblock, err := bc.mainDB.GetHashBlock(bc.MyChain[i].Hash[:])
				if err != nil {
					log.Printf("Failed to get old block at height %d: %v", i, err)
					return
				}
				bc.UNDoTxn(&oldblock.Txn)
				log.Printf("Rolled back transaction at height %d", i)
			}

			// Resize MyChain to the fork point (height)
			bc.MyChain = bc.MyChain[:height]
			log.Printf("Resized chain to fork point at height %d", height)

			// Add new blocks to our chain and process their transactions
			log.Printf("Adding %d new blocks to chain", newBlock.Height-height+1)
			for i := height; i <= newBlock.Height; i++ {
				if block, exists := newchain[i]; exists {
					// Add block to our chain
					bc.MyChain = append(bc.MyChain, &Chain{Hash: block.Hash(), PrvHash: block.PreHash})

					// Process transactions
					bc.DoTxn(&block.Txn)

					// Update database
					blockHash := block.Hash()
					err := bc.mainDB.InsertHashBlock(&blockHash, block)
					if err != nil {
						log.Printf("Failed to insert block %x at height %d: %v",
							blockHash, block.Height, err)
						return
					}
					log.Printf("Added block %x at height %d to chain", blockHash, i)
				}
			}

			// Update tip in database
			tipHash := newBlock.Hash()
			err := bc.mainDB.InsertTipHash(&tipHash)
			if err != nil {
				log.Printf("Failed to update tip hash: %v", err)
				return
			}
			log.Printf("Chain tip changed to %x at height %d", tipHash, newBlock.Height)
			return
		}

		if height <= 1 {
			log.Printf("Reached genesis block height without finding fork point")
			return
		}
	}
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
			bc.P2PChan <- &p2p.P2PBlock{Block: *result.block, Sender: selectedPeer.String()}
		}
	case <-ctx.Done():
		log.Printf("Timeout waiting for tip from peer %s", selectedPeer)
	}
}
