package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/nanlour/da/block"
)

const (
	// Protocol identifiers
	blockByHashProtocol = "/blockchain/getblockbyhash/1.0.0"
	getTipProtocol      = "/blockchain/gettip/1.0.0"
)

// Request/response types
type BlockByHashRequest struct {
	Hash [32]byte `json:"hash"`
}

type BlockResponse struct {
	Block *block.Block `json:"block"`
	Error string       `json:"error,omitempty"`
}

// setupProtocols initializes all protocol handlers
func (s *Service) setupProtocols() {
	// Register protocol handlers
	s.host.SetStreamHandler(protocol.ID(blockByHashProtocol), s.handleBlockByHashRequest)
	s.host.SetStreamHandler(protocol.ID(getTipProtocol), s.handleGetTipRequest)
}

// handleBlockByHashRequest processes incoming block-by-hash requests
func (s *Service) handleBlockByHashRequest(stream network.Stream) {
	defer stream.Close()

	// Read the request
	var request BlockByHashRequest
	if err := json.NewDecoder(stream).Decode(&request); err != nil {
		sendErrorResponse(stream, "Failed to decode request")
		return
	}

	// Process the request using the blockchain
	var response BlockResponse

	// Get the block from the blockchain
	block, err := s.blockchain.GetBlockByHash(request.Hash[:])
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Block = block
	}

	// Send the response
	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending response: %s\n", err)
		return
	}
}

// handleGetTipRequest processes incoming tip requests
func (s *Service) handleGetTipRequest(stream network.Stream) {
	defer stream.Close()

	// Process the request using the blockchain
	var response BlockResponse

	// Get the tip from the blockchain
	tipHash, err := s.blockchain.GetTipHash()
	if err != nil {
		response.Error = err.Error()
		json.NewEncoder(stream).Encode(response)
		return
	}

	block, err := s.blockchain.GetBlockByHash(tipHash)
	if err != nil {
		response.Error = err.Error()
		json.NewEncoder(stream).Encode(response)
		return
	}

	// Convert slice to fixed-size array
	var hashArray [32]byte
	copy(hashArray[:], tipHash)

	response.Block = block

	// Send the response
	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending response: %s\n", err)
		return
	}
}

// Helper function to send an error response
func sendErrorResponse(stream network.Stream, errMsg string) {
	json.NewEncoder(stream).Encode(map[string]string{"error": errMsg})
}

// GetBlockByHash requests a block from the P2P network by its hash
func (s *Service) GetBlockByHash(hash [32]byte, peerID peer.ID) (*block.Block, error) {
	peers := s.Peers()
	if len(peers) == 0 {
		return nil, fmt.Errorf("no connected peers")
	}

	// Create a new stream
	stream, err := s.host.NewStream(s.ctx, peerID, protocol.ID(blockByHashProtocol))
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	// Send request
	request := BlockByHashRequest{Hash: hash}
	if err := json.NewEncoder(stream).Encode(request); err != nil {
		return nil, err
	}

	// Read response
	var response BlockResponse
	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, err
	}

	// Check for error in response
	if response.Error != "" {
		return nil, fmt.Errorf("peer error: %s", response.Error)
	}

	return response.Block, nil
}

// GetTip requests the current blockchain tip from the P2P network
func (s *Service) GetTip(peerID peer.ID) (*block.Block, error) {
	peers := s.Peers()
	if len(peers) == 0 {
		return nil, fmt.Errorf("no connected peers")
	}

	// Create a new stream
	stream, err := s.host.NewStream(s.ctx, peerID, protocol.ID(getTipProtocol))
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	// No data needed for tip request, just close the write side
	if err := stream.CloseWrite(); err != nil {
		return nil, err
	}

	// Read response
	var response BlockResponse
	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, err
	}

	// Check for error in response
	if response.Error != "" {
		return nil, fmt.Errorf("peer error: %s", response.Error)
	}

	return response.Block, nil
}
