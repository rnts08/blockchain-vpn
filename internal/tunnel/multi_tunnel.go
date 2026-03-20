package tunnel

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

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

// ReconnectConfig holds the configuration for automatic reconnection.
type ReconnectConfig struct {
	Enabled      bool
	MaxAttempts  int           // 0 = infinite
	BaseInterval time.Duration // Starting interval between retries
	MaxInterval  time.Duration // Maximum interval cap
}

// tunnelParams holds the parameters needed to establish and reconnect a tunnel.
type tunnelParams struct {
	interfaceName string
	clientCfg     *config.ClientConfig
	secCfg        *config.SecurityConfig
	localKey      *btcec.PrivateKey
	peerPubKey    *btcec.PublicKey
	endpointAddr  string
	expectations  ClientSecurityExpectations
	pricingParams *PricingParams
}

// MultiTunnelManager tracks and orchestrates multiple concurrent client VPN connections.
type MultiTunnelManager struct {
	tunnels       map[string]*ActiveTunnel
	reconnectInfo map[string]*tunnelParams // Stored params for reconnection
	mu            sync.RWMutex
}

// NewMultiTunnelManager creates a new empty tunnel manager.
func NewMultiTunnelManager() *MultiTunnelManager {
	return &MultiTunnelManager{
		tunnels:       make(map[string]*ActiveTunnel),
		reconnectInfo: make(map[string]*tunnelParams),
	}
}

const defaultConnectTimeout = 30 * time.Second

// Add starts a new tunnel connection in a separate goroutine and tracks it.
// The connection attempt has a 30-second timeout to prevent indefinite blocking.
func (m *MultiTunnelManager) Add(
	id string,
	interfaceName string,
	clientCfg *config.ClientConfig,
	secCfg *config.SecurityConfig,
	localKey *btcec.PrivateKey,
	peerPubKey *btcec.PublicKey,
	endpointAddr string,
	expectations ClientSecurityExpectations,
	spendingMgr *SpendingManager, // optional, can be nil
	pricingParams *PricingParams, // optional, for time/data billing
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tunnels[id]; exists {
		return fmt.Errorf("tunnel with ID %s already running", id)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectTimeout)
	tunnel := &ActiveTunnel{
		ID:        id,
		ctx:       ctx,
		cancel:    cancel,
		done:      make(chan struct{}),
		Interface: interfaceName,
	}

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
			spendingMgr,
			pricingParams,
		)
		if tunnel.err != nil {
			log.Printf("[MultiTunnel] Tunnel %s stopped with error: %v", id, tunnel.err)
		} else {
			log.Printf("[MultiTunnel] Tunnel %s stopped cleanly.", id)
		}
	}()

	return nil
}

// AddWithReconnect starts a tunnel with automatic reconnection on disconnect.
// It stores the connection parameters for reconnection attempts.
func (m *MultiTunnelManager) AddWithReconnect(
	id string,
	interfaceName string,
	clientCfg *config.ClientConfig,
	secCfg *config.SecurityConfig,
	localKey *btcec.PrivateKey,
	peerPubKey *btcec.PublicKey,
	endpointAddr string,
	expectations ClientSecurityExpectations,
	spendingMgr *SpendingManager,
	pricingParams *PricingParams,
) error {
	reconnectCfg := ReconnectConfig{
		Enabled:      clientCfg.AutoReconnectEnabled,
		MaxAttempts:  clientCfg.AutoReconnectMaxAttempts,
		BaseInterval: parseAutoReconnectInterval(clientCfg.AutoReconnectInterval),
		MaxInterval:  parseAutoReconnectInterval(clientCfg.AutoReconnectMaxInterval),
	}

	if reconnectCfg.BaseInterval == 0 {
		reconnectCfg.BaseInterval = 5 * time.Second
	}
	if reconnectCfg.MaxInterval == 0 {
		reconnectCfg.MaxInterval = 5 * time.Minute
	}

	m.mu.Lock()
	m.reconnectInfo[id] = &tunnelParams{
		interfaceName: interfaceName,
		clientCfg:     clientCfg,
		secCfg:        secCfg,
		localKey:      localKey,
		peerPubKey:    peerPubKey,
		endpointAddr:  endpointAddr,
		expectations:  expectations,
		pricingParams: pricingParams,
	}
	m.mu.Unlock()

	err := m.Add(id, interfaceName, clientCfg, secCfg, localKey, peerPubKey, endpointAddr, expectations, spendingMgr, pricingParams)
	if err != nil {
		m.mu.Lock()
		delete(m.reconnectInfo, id)
		m.mu.Unlock()
		return err
	}

	if reconnectCfg.Enabled {
		go m.reconnectLoop(id, reconnectCfg, spendingMgr)
	}

	return nil
}

// reconnectLoop monitors a tunnel and attempts to reconnect on disconnect.
func (m *MultiTunnelManager) reconnectLoop(id string, cfg ReconnectConfig, spendingMgr *SpendingManager) {
	attempt := 0
	interval := cfg.BaseInterval

	for {
		m.mu.RLock()
		_, exists := m.tunnels[id]
		reconnectInfo := m.reconnectInfo[id]
		m.mu.RUnlock()

		if !exists {
			if !cfg.Enabled || (cfg.MaxAttempts > 0 && attempt >= cfg.MaxAttempts) {
				log.Printf("[MultiTunnel] Reconnection disabled or max attempts reached for %s", id)
				m.mu.Lock()
				delete(m.reconnectInfo, id)
				m.mu.Unlock()
				return
			}

			attempt++
			log.Printf("[MultiTunnel] Attempting to reconnect %s (attempt %d/%v) after %v...",
				id, attempt, cfg.MaxAttempts, interval)

			m.mu.RLock()
			params := reconnectInfo
			m.mu.RUnlock()

			if params == nil {
				log.Printf("[MultiTunnel] No reconnection info available for %s", id)
				return
			}

			cfgCopy := *params.clientCfg
			cfgCopy.InterfaceName = params.interfaceName

			err := m.Add(id, params.interfaceName, &cfgCopy, params.secCfg, params.localKey,
				params.peerPubKey, params.endpointAddr, params.expectations, spendingMgr, params.pricingParams)
			if err != nil {
				log.Printf("[MultiTunnel] Failed to start reconnection for %s: %v", id, err)
				m.mu.Lock()
				delete(m.reconnectInfo, id)
				m.mu.Unlock()
				return
			}

			interval = time.Duration(math.Min(float64(interval*2), float64(cfg.MaxInterval)))
			if interval < cfg.BaseInterval {
				interval = cfg.BaseInterval
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// parseAutoReconnectInterval parses a duration string like "5s", "30s", "5m".
func parseAutoReconnectInterval(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("[MultiTunnel] Invalid auto-reconnect interval %q: %v", s, err)
		return 0
	}
	return d
}

// Cancel stops a specific tunnel by its ID and removes its reconnection info.
func (m *MultiTunnelManager) Cancel(id string) {
	m.mu.RLock()
	t, ok := m.tunnels[id]
	m.mu.RUnlock()

	if ok && t.cancel != nil {
		log.Printf("[MultiTunnel] Cancelling tunnel %s...", id)
		t.cancel()
		<-t.done // Wait for it to tear down
	}

	m.mu.Lock()
	delete(m.reconnectInfo, id)
	m.mu.Unlock()
}

// CancelAll shuts down all active tunnels concurrently and waits for them to exit.
func (m *MultiTunnelManager) CancelAll() {
	m.mu.Lock()
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
	m.tunnels = make(map[string]*ActiveTunnel)
	m.reconnectInfo = make(map[string]*tunnelParams)
	m.mu.Unlock()
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

// ActiveCount returns the number of currently active tunnels.
func (m *MultiTunnelManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tunnels)
}
