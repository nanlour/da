package p2p

import (
	"context"
	"encoding/json"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/nanlour/da/src/block"
)

const (
	// PubSub topics
	blockTopic = "blocks"
	txTopic    = "transactions"
)

// PubSubManager manages pubsub functionality
type PubSubManager struct {
	ps         *pubsub.PubSub
	blockTopic *pubsub.Topic
	blockSub   *pubsub.Subscription
	txTopic    *pubsub.Topic
	txSub      *pubsub.Subscription
	ctx        context.Context
	blockchain BlockchainInterface
}

// initPubSub initializes the PubSub system
func (s *Service) initPubSub() error {
	// Create a new PubSub service using GossipSub
	ps, err := pubsub.NewGossipSub(s.ctx, s.host)
	if err != nil {
		return err
	}

	// Join the block topic
	blockTopic, err := ps.Join(blockTopic)
	if err != nil {
		return err
	}

	// Subscribe to the block topic
	blockSub, err := blockTopic.Subscribe()
	if err != nil {
		return err
	}

	// Join the transaction topic
	txTopic, err := ps.Join(txTopic)
	if err != nil {
		return err
	}

	// Subscribe to the transaction topic
	txSub, err := txTopic.Subscribe()
	if err != nil {
		return err
	}

	s.pubsubMgr = &PubSubManager{
		ps:         ps,
		blockTopic: blockTopic,
		blockSub:   blockSub,
		txTopic:    txTopic,
		txSub:      txSub,
		ctx:        s.ctx,
		blockchain: s.blockchain,
	}

	// Start processing messages
	go s.pubsubMgr.processBlockMessages()
	go s.pubsubMgr.processTxMessages()

	return nil
}

// BroadcastBlock broadcasts a block to the network
func (s *Service) BroadcastBlock(block *block.Block) error {
	if s.pubsubMgr == nil || s.pubsubMgr.blockTopic == nil {
		return fmt.Errorf("pubsub not initialized")
	}

	blockData, err := json.Marshal(block)
	if err != nil {
		return err
	}

	return s.pubsubMgr.blockTopic.Publish(s.ctx, blockData)
}

// BroadcastTransaction broadcasts a transaction to the network
func (s *Service) BroadcastTransaction(tx *block.Transaction) error {
	if s.pubsubMgr == nil || s.pubsubMgr.txTopic == nil {
		return fmt.Errorf("pubsub not initialized")
	}

	txData, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return s.pubsubMgr.txTopic.Publish(s.ctx, txData)
}

// Process incoming block messages
func (pm *PubSubManager) processBlockMessages() {
	for {
		msg, err := pm.blockSub.Next(pm.ctx)
		if err != nil {
			// Context canceled or subscription closed
			return
		}

		// Get the sender's peer ID
		sender := msg.ReceivedFrom.String()

		var block block.Block
		if err := json.Unmarshal(msg.Data, &block); err != nil {
			fmt.Printf("Error unmarshaling block from %s: %s\n", sender, err)
			continue
		}

		// Add the block to the blockchain
		if err := pm.blockchain.AddBlock(&block); err != nil {
			fmt.Printf("Error adding block from %s to blockchain: %s\n", sender, err)
			continue
		}

		fmt.Printf("Received and added new block from %s: %x\n", sender, block)
	}
}

// Process incoming transaction messages
func (pm *PubSubManager) processTxMessages() {
	for {
		msg, err := pm.txSub.Next(pm.ctx)
		if err != nil {
			// Context canceled or subscription closed
			return
		}

		// Get the sender's peer ID
		sender := msg.ReceivedFrom.String()

		var tx block.Transaction
		if err := json.Unmarshal(msg.Data, &tx); err != nil {
			fmt.Printf("Error unmarshaling transaction from %s: %s\n", sender, err)
			continue
		}

		// Add the txn to mempool
		if err := pm.blockchain.AddTxn(&tx); err != nil {
			fmt.Printf("Error adding block from %s to blockchain: %s\n", sender, err)
			continue
		}

		// Process the transaction (add to mempool, etc.)
		fmt.Printf("Received new transaction from %s: %x\n", sender, tx.Hash())
	}
}
