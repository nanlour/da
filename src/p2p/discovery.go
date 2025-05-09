package p2p

import (
	"fmt"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = 10 * time.Second

// DiscoveryServiceTag is used in our mDNS advertisements to discover other peers.
const DiscoveryServiceTag = "da-p2p-discovery"

// setupDiscovery configures peer discovery mechanisms
func (s *Service) setupDiscovery() error {
	// Setup mDNS discovery
	if err := s.setupMDNS(); err != nil {
		return err
	}

	// Setup DHT discovery
	if err := s.setupDHT(); err != nil {
		return err
	}

	return nil
}

// setupMDNS configures a new mDNS discovery service and attaches it to the libp2p Host.
func (s *Service) setupMDNS() error {
	// Create a new mDNS service
	discovery := mdns.NewMdnsService(
		s.host,
		DiscoveryServiceTag,
		&discoveryNotifee{s: s},
	)

	return discovery.Start()
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	s *Service
}

// HandlePeerFound connects to peers discovered via mDNS
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("Discovered new peer %s\n", pi.ID.String())

	// Don't connect to self
	if pi.ID == n.s.host.ID() {
		return
	}

	// Connect to the newly discovered peer
	err := n.s.host.Connect(n.s.ctx, pi)
	if err != nil {
		fmt.Printf("Error connecting to peer %s: %s\n", pi.ID.String(), err)
		return
	}

	n.s.peersMu.Lock()
	n.s.peers[pi.ID] = pi
	n.s.peersMu.Unlock()

	fmt.Printf("%s Connected to peer: %s\n", n.s.host.ID(), pi.ID.String())
}

// setupDHT initializes the DHT for peer discovery
func (s *Service) setupDHT() error {
	// Create DHT server mode
	kdht, err := dht.New(s.ctx, s.host, dht.Mode(dht.ModeServer))
	if err != nil {
		return err
	}

	// Bootstrap the DHT
	if err = kdht.Bootstrap(s.ctx); err != nil {
		return err
	}

	// Add DHT to service for later use
	s.dht = kdht

	// Connect to bootstrap nodes if specified
	if len(s.bootstrapPeers) > 0 {
		go s.connectToBootstrapPeers()
	}

	return nil
}

// connectToBootstrapPeers tries to connect to the bootstrap peers
func (s *Service) connectToBootstrapPeers() {
	for _, peerAddr := range s.bootstrapPeers {
		pi, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			fmt.Printf("Error parsing bootstrap peer address: %s\n", err)
			continue
		}

		err = s.host.Connect(s.ctx, *pi)
		if err != nil {
			fmt.Printf("Failed to connect to bootstrap node %s: %s\n", pi.ID, err)
		} else {
			fmt.Printf("Connected to bootstrap node: %s\n", pi.ID)

			s.peersMu.Lock()
			s.peers[pi.ID] = *pi
			s.peersMu.Unlock()
		}
	}
}

// AddBootstrapPeer adds a peer multiaddress to the bootstrap list
func (s *Service) AddBootstrapPeer(addr string) error {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return err
	}

	s.bootstrapPeers = append(s.bootstrapPeers, maddr)
	return nil
}
