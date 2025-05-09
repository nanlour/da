package p2p

import (
	"context"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/nanlour/da/block"
)

// Service represents the P2P networking service
type Service struct {
	host           host.Host
	ctx            context.Context
	cancel         context.CancelFunc
	peersMu        sync.RWMutex
	peers          map[peer.ID]peer.AddrInfo
	pubsubMgr      *PubSubManager
	blockchain     BlockchainInterface
	dht            *dht.IpfsDHT
	bootstrapPeers []multiaddr.Multiaddr
}

// BlockchainInterface defines the methods required from the blockchain
type BlockchainInterface interface {
	AddBlock(block *block.Block) error
	AddTxn(block *block.Transaction) error
	GetBlockByHash(hash []byte) (*block.Block, error)
	GetTipBlock() (*block.Block, error)    
}

// NewService creates and initializes a new P2P service
func NewService(listenAddr string, blockchain BlockchainInterface) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Parse the multiaddress
	addr, err := multiaddr.NewMultiaddr(listenAddr)
	if err != nil {
		cancel()
		return nil, err
	}

	// Create a new libp2p Host
	h, err := libp2p.New(
		libp2p.ListenAddrs(addr),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	s := &Service{
		host:           h,
		ctx:            ctx,
		cancel:         cancel,
		peers:          make(map[peer.ID]peer.AddrInfo),
		blockchain:     blockchain,
		bootstrapPeers: []multiaddr.Multiaddr{},
	}

	// Set up protocol handlers
	s.setupProtocols()

	return s, nil
}

// Start starts the P2P service
func (s *Service) Start() error {
	fmt.Printf("P2P service started. Host ID: %s\n", s.host.ID().String())
	fmt.Println("Listening on:")
	for _, addr := range s.host.Addrs() {
		fmt.Printf("  %s/p2p/%s\n", addr, s.host.ID().String())
	}

	// Initialize pubsub
	if err := s.initPubSub(); err != nil {
		return err
	}

	// Initialize peer discovery
	if err := s.setupDiscovery(); err != nil {
		return fmt.Errorf("failed to setup discovery: %w", err)
	}

	return nil
}

// Stop gracefully stops the P2P service
func (s *Service) Stop() error {
	s.cancel()
	return s.host.Close()
}

// Connect attempts to connect to a peer at the given address
func (s *Service) Connect(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}

	addrInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	if err := s.host.Connect(s.ctx, *addrInfo); err != nil {
		return err
	}

	s.peersMu.Lock()
	s.peers[addrInfo.ID] = *addrInfo
	s.peersMu.Unlock()

	fmt.Printf("Connected to peer: %s\n", addrInfo.ID.String())
	return nil
}

// Peers returns a list of connected peers
func (s *Service) Peers() []peer.ID {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()

	peers := make([]peer.ID, 0, len(s.peers))
	for id := range s.peers {
		peers = append(peers, id)
	}
	return peers
}
