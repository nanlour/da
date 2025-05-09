package rpc

import (
	"fmt"
	"log"
	"net"
	netRPC "net/rpc"
	"sync/atomic"
)

// RPCServer represents the blockchain RPC server
type RPCServer struct {
	server    *netRPC.Server
	listener  net.Listener
	port      int
	isRunning int32
}

// NewRPCServer creates and returns a new RPCServer instance
func NewRPCServer(port int) *RPCServer {
	return &RPCServer{
		server:    netRPC.NewServer(),
		port:      port,
		isRunning: 0,
	}
}

// Start initializes and starts the RPC server
func (s *RPCServer) Start(blockchain BlockchainInterface) error {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 0, 1) {
		return fmt.Errorf("RPC server is already running")
	}

	// Register the blockchain service
	blockchainService := &BlockchainService{blockchain: blockchain}
	if err := s.server.RegisterName("BlockchainService", blockchainService); err != nil {
		return fmt.Errorf("failed to register BlockchainService: %v", err)
	}

	// Create a TCP listener
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start RPC listener on port %d: %v", s.port, err)
	}

	log.Printf("RPC server started on port %d", s.port)

	// Accept connections in a goroutine
	go s.acceptConnections()

	return nil
}

// acceptConnections handles incoming RPC connections
func (s *RPCServer) acceptConnections() {
	for atomic.LoadInt32(&s.isRunning) == 1 {
		conn, err := s.listener.Accept()
		if err != nil {
			// If server is stopping, this is expected
			if atomic.LoadInt32(&s.isRunning) == 0 {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Handle the connection in a new goroutine
		go s.server.ServeConn(conn)
	}
}

// Stop shuts down the RPC server
func (s *RPCServer) Stop() error {
	if !atomic.CompareAndSwapInt32(&s.isRunning, 1, 0) {
		return fmt.Errorf("RPC server is not running")
	}

	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("error stopping RPC server: %v", err)
	}

	log.Println("RPC server stopped")
	return nil
}
