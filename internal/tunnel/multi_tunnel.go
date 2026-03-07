package tunnel

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/btcsuite/btcd/btcec/v2"

	"blockchain-vpn/internal/config"
)

// ActiveTunnel represents a single running VPN connection to a provider.
type ActiveTunnel struct {
	ID        string
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	err       error
	Interface string // The TUN interface name, e.g. bcvpn0
}

// MultiTunnelManager tracks and orchestrates multiple concurrent client VPN connections.
type MultiTunnelManager struct {
	tunnels map[string]*ActiveTunnel
	mu      sync.RWMutex
}

// NewMultiTunnelManager creates a new empty tunnel manager.
func NewMultiTunnelManager() *MultiTunnelManager {
	return &MultiTunnelManager{
		tunnels: make(map[string]*ActiveTunnel),
	}
}

// Add starts a new tunnel connection in a separate goroutine and tracks it.
func (m *MultiTunnelManager) Add(
	id string,
	interfaceName string,
	clientCfg *config.ClientConfig,
	secCfg *config.SecurityConfig,
	localKey *btcec.PrivateKey,
	peerPubKey *btcec.PublicKey,
	endpointAddr string,
	expectations ClientSecurityExpectations,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tunnels[id]; exists {
		return fmt.Errorf("tunnel with ID %s already running", id)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tunnel := &ActiveTunnel{
		ID:        id,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		Interface: interfaceName,
	}

	// We create a copy of the client config to override the interface name safely for this connection
	cfgCopy := *clientCfg
	cfgCopy.InterfaceName = interfaceName

	m.tunnels[id] = tunnel

	go func() {
		defer close(tunnel.done)
		defer func() {
			m.mu.Lock()
			delete(m.tunnels, id)
			m.mu.Unlock()
		}()

		log.Printf("[MultiTunnel] Starting tunnel %s on %s...", id, interfaceName)
		tunnel.err = ConnectToProvider(
			tunnel.ctx,
			&cfgCopy,
			secCfg,
			localKey,
			peerPubKey,
			endpointAddr,
			expectations,
		)
		if tunnel.err != nil {
			log.Printf("[MultiTunnel] Tunnel %s stopped with error: %v", id, tunnel.err)
		} else {
			log.Printf("[MultiTunnel] Tunnel %s stopped cleanly.", id)
		}
	}()

	return nil
}

// Cancel stops a specific tunnel by its ID.
func (m *MultiTunnelManager) Cancel(id string) {
	m.mu.RLock()
	t, ok := m.tunnels[id]
	m.mu.RUnlock()

	if ok && t.cancel != nil {
		log.Printf("[MultiTunnel] Cancelling tunnel %s...", id)
		t.cancel()
		<-t.done // Wait for it to tear down
	}
}

// CancelAll shuts down all active tunnels concurrently and waits for them to exit.
func (m *MultiTunnelManager) CancelAll() {
	m.mu.RLock()
	var wg sync.WaitGroup
	for id, t := range m.tunnels {
		wg.Add(1)
		go func(tID string, tunnel *ActiveTunnel) {
			defer wg.Done()
			if tunnel.cancel != nil {
				log.Printf("[MultiTunnel] Cancelling tunnel %s...", tID)
				tunnel.cancel()
				<-tunnel.done
			}
		}(id, t)
	}
	m.mu.RUnlock()
	wg.Wait()
}

// List returns a list of active tunnel IDs and their assigned interfaces.
func (m *MultiTunnelManager) List() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make(map[string]string, len(m.tunnels))
	for id, t := range m.tunnels {
		list[id] = t.Interface
	}
	return list
}
