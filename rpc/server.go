package rpc

import (
    "fmt"
    "log"
    "net"
    netRPC "net/rpc"
)

// RPCServer represents the blockchain RPC server
type RPCServer struct {
    server     *netRPC.Server
    listener   net.Listener
    port       int
    isRunning  bool
}

// NewRPCServer creates and returns a new RPCServer instance
func NewRPCServer(port int) *RPCServer {
    return &RPCServer{
        server:    netRPC.NewServer(),
        port:      port,
        isRunning: false,
    }
}

// Start initializes and starts the RPC server
func (s *RPCServer) Start() error {
    if s.isRunning {
        return fmt.Errorf("RPC server is already running")
    }

    // Register the blockchain service
    blockchainService := &BlockchainService{}
    if err := s.server.RegisterName("BlockchainService", blockchainService); err != nil {
        return fmt.Errorf("failed to register BlockchainService: %v", err)
    }

    // Create a TCP listener
    var err error
    s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.port))
    if err != nil {
        return fmt.Errorf("failed to start RPC listener on port %d: %v", s.port, err)
    }

    s.isRunning = true
    log.Printf("RPC server started on port %d", s.port)

    // Accept connections in a goroutine
    go s.acceptConnections()

    return nil
}

// acceptConnections handles incoming RPC connections
func (s *RPCServer) acceptConnections() {
    for s.isRunning {
        conn, err := s.listener.Accept()
        if err != nil {
            // If server is stopping, this is expected
            if !s.isRunning {
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
    if !s.isRunning {
        return fmt.Errorf("RPC server is not running")
    }

    s.isRunning = false
    if err := s.listener.Close(); err != nil {
        return fmt.Errorf("error stopping RPC server: %v", err)
    }

    log.Println("RPC server stopped")
    return nil
}