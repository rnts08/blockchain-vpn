package tunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/config"
	"blockchain-vpn/internal/transport"
	"blockchain-vpn/internal/util"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/songgao/water"
)

// createTunInterface creates and configures a new TUN interface.
func createTunInterface(ifaceName string, ip string, subnetMask string) (*water.Interface, error) {
	iface, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: platformSpecificParams(ifaceName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	log.Printf("TUN interface %s created", iface.Name())

	if err := configureTunInterface(iface.Name(), ip, subnetMask); err != nil {
		return nil, err
	}

	return iface, nil
}

// IPPool manages a pool of dynamic IP addresses.
type IPPool struct {
	baseIP net.IP
	used   map[string]bool
	mu     sync.Mutex
}

func NewIPPool(gatewayIP string) *IPPool {
	return &IPPool{
		baseIP: net.ParseIP(gatewayIP),
		used:   make(map[string]bool),
	}
}

// Allocate finds the next available IP in the /24 subnet (IPv4) or /120 (IPv6).
func (p *IPPool) Allocate() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ip4 := p.baseIP.To4()
	if ip4 != nil {
		// IPv4 allocation strategy: iterate 2 to 254 in a /24 subnet
		res := make(net.IP, len(ip4))
		copy(res, ip4)
		for i := 2; i < 255; i++ {
			res[3] = byte(i)
			ipStr := res.String()
			if ipStr == p.baseIP.String() {
				continue
			}
			if !p.used[ipStr] {
				p.used[ipStr] = true
				return res, nil
			}
		}
	} else {
		// IPv6 allocation strategy: iterate last byte 2 to 254 in a /120 range
		res := make(net.IP, len(p.baseIP))
		copy(res, p.baseIP)
		for i := 2; i < 255; i++ {
			res[15] = byte(i)
			ipStr := res.String()
			if ipStr == p.baseIP.String() {
				continue
			}
			if !p.used[ipStr] {
				p.used[ipStr] = true
				return res, nil
			}
		}
	}
	return nil, fmt.Errorf("ip pool exhausted")
}

func (p *IPPool) Release(ip net.IP) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.used, ip.String())
}

// ClientMap maps IP addresses to active connections.
type ClientMap struct {
	m  map[string]*clientSession
	mu sync.RWMutex
}

// AddSession atomically registers a session under ip, respecting maxConsumers.
// Returns false (and does NOT register the session) if the limit is already reached.
func (c *ClientMap) AddSession(ip net.IP, session *clientSession, maxConsumers int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if maxConsumers > 0 && len(c.m) >= maxConsumers {
		return false
	}
	c.m[ip.String()] = session
	sessionOpened()
	return true
}

// RemoveSession unregisters the session for ip and decrements the active-session counter.
func (c *ClientMap) RemoveSession(ip net.IP) {
	c.mu.Lock()
	delete(c.m, ip.String())
	c.mu.Unlock()
	sessionClosed()
}

// StartProviderServer sets up the TUN interface and listens for TLS connections.
func StartProviderServer(ctx context.Context, cfg *config.ProviderConfig, sec *config.SecurityConfig, privKey *btcec.PrivateKey, authManager *auth.AuthManager) error {
	if err := EnsureElevatedPrivileges(); err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("provider requires automatic networking privileges: %w", err)
	}
	metricsToken := ""
	if sec != nil {
		metricsToken = strings.TrimSpace(sec.MetricsAuthToken)
	}
	startMetricsServer(cfg.MetricsListenAddr, metricsToken)
	setProviderRunning(true)
	defer setProviderRunning(false)

	if err := applyProviderIsolation(cfg.IsolationMode); err != nil {
		log.Printf("Warning: could not apply provider isolation mode %q: %v", cfg.IsolationMode, err)
	} else if cfg.IsolationMode != "" && cfg.IsolationMode != "none" {
		log.Printf("Provider isolation mode enabled: %s", cfg.IsolationMode)
	}

	certLifetime := time.Duration(cfg.CertLifetimeHours) * time.Hour
	certRotateBefore := time.Duration(cfg.CertRotateBeforeHours) * time.Hour
	tlsPolicy := TLSPolicy{}
	var err error
	var customCiphers []string
	if sec != nil {
		customCiphers = sec.TlsCustomCipherSuites
	}
	if sec != nil {
		tlsPolicy, err = ResolveTLSPolicy(sec.TLSMinVersion, sec.TLSProfile, customCiphers)
		if err != nil {
			recordRuntimeError(err)
			return fmt.Errorf("failed to resolve TLS policy: %w", err)
		}
	} else {
		tlsPolicy, _ = ResolveTLSPolicy("", "", nil)
	}
	tlsConfig, err := buildRotatingServerTLSConfig(ctx, privKey, certLifetime, certRotateBefore, tlsPolicy)
	if err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("failed to generate server TLS config: %w", err)
	}

	policy, err := loadAccessPolicy(cfg.AllowlistFile, cfg.DenylistFile)
	if err != nil {
		return fmt.Errorf("failed to load provider access policy: %w", err)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort)
	listener, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("failed to start TLS listener: %w", err)
	}
	defer listener.Close()
	log.Printf("Provider server listening on %s", listenAddr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	iface, err := createTunInterface(cfg.InterfaceName, cfg.TunIP, cfg.TunSubnet)
	if err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("failed to create provider TUN interface: %w", err)
	}
	defer iface.Close()

	if cfg.EnableEgressNAT {
		natCleanup, err := setupProviderEgressNAT(iface.Name(), cfg.TunIP, cfg.TunSubnet, cfg.NATOutboundInterface)
		if err != nil {
			recordRuntimeError(err)
			log.Printf("Warning: failed to configure provider egress NAT: %v", err)
		} else if natCleanup != nil {
			defer natCleanup()
			log.Printf("Provider egress NAT enabled for %s", iface.Name())
		}
	}

	startProviderHealthChecks(ctx, cfg, iface.Name(), listenAddr)

	// Initialize IP Pool and Client Map
	ipPool := NewIPPool(cfg.TunIP)
	clients := &ClientMap{m: make(map[string]*clientSession)}
	limitBytesPerSec, err := parseBandwidthLimit(cfg.BandwidthLimit)
	if err != nil {
		log.Printf("Warning: invalid provider bandwidth_limit %q: %v (disabling throttle)", cfg.BandwidthLimit, err)
		limitBytesPerSec = 0
	}

	// Start the central TUN reader goroutine
	// This reads packets from the TUN interface and routes them to the correct client connection.
	go readTunLoop(iface, clients)

	if cfg.WebSocketFallbackPort > 0 {
		wsAddr := fmt.Sprintf("0.0.0.0:%d", cfg.WebSocketFallbackPort)
		go func() {
			log.Printf("Provider WebSocket server listening on %s", wsAddr)
			if err := transport.StartWSServer(ctx, wsAddr, tlsConfig, func(conn net.Conn, r *http.Request) {
				handleClient(ctx, conn, cfg, sec, privKey, authManager, ipPool, clients, limitBytesPerSec, iface, policy, r)
			}); err != nil {
				log.Printf("WebSocket server error: %v", err)
			}
		}()
	}

	acceptBackoff := 100 * time.Millisecond
	acceptRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("Provider server shutting down.")
				return nil
			default:
				recordRuntimeError(err)
				log.Printf("Failed to accept connection: %v", err)
				jitter := 0.8 + acceptRand.Float64()*0.4
				sleep := time.Duration(float64(acceptBackoff) * jitter)
				time.Sleep(sleep)
				if acceptBackoff < 2*time.Second {
					acceptBackoff *= 2
					if acceptBackoff > 2*time.Second {
						acceptBackoff = 2 * time.Second
					}
				}
				continue
			}
		}
		acceptBackoff = 100 * time.Millisecond

		handleClient(ctx, conn, cfg, sec, privKey, authManager, ipPool, clients, limitBytesPerSec, iface, policy, nil)
	}
}

func handleClient(ctx context.Context, conn net.Conn, cfg *config.ProviderConfig, sec *config.SecurityConfig, privKey *btcec.PrivateKey, authManager *auth.AuthManager, ipPool *IPPool, clients *ClientMap, limitBytesPerSec int64, iface *water.Interface, policy *accessPolicy, r *http.Request) {
	// The handshake is already complete from Accept() or WSS upgrade.
	// We verify the peer's certificate.
	var clientPubKey *btcec.PublicKey
	var err error

	if tlsConn, ok := conn.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			log.Printf("Connection from %s rejected: handshake failed: %v", conn.RemoteAddr(), err)
			recordEvent("provider", "reject_handshake", conn.RemoteAddr().String())
			conn.Close()
			recordRuntimeError(fmt.Errorf("client handshake failed: %w", err))
			return
		}
		state := tlsConn.ConnectionState()
		if len(state.PeerCertificates) == 0 {
			log.Printf("Connection from %s rejected: no client certificate provided", conn.RemoteAddr())
			recordEvent("provider", "reject_no_cert", conn.RemoteAddr().String())
			conn.Close()
			recordRuntimeError(fmt.Errorf("client connection rejected: no certificate"))
			return
		}
		clientPubKey, err = certToBTCECPubKey(state.PeerCertificates[0])
		if err != nil {
			log.Printf("Connection from %s rejected: %v", conn.RemoteAddr(), err)
			recordEvent("provider", "reject_bad_cert", conn.RemoteAddr().String())
			conn.Close()
			recordRuntimeError(err)
			return
		}
	} else if r != nil && r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		clientPubKey, err = certToBTCECPubKey(r.TLS.PeerCertificates[0])
		if err != nil {
			log.Printf("WebSocket connection from %s rejected: %v", conn.RemoteAddr(), err)
			conn.Close()
			return
		}
	}

	if clientPubKey == nil {
		log.Printf("Connection from %s rejected: could not determine client identity", conn.RemoteAddr())
		conn.Close()
		return
	}

	if err := policy.check(clientPubKey); err != nil {
		log.Printf("Connection from %s rejected by policy: %v", conn.RemoteAddr(), err)
		recordEvent("provider", "reject_policy", conn.RemoteAddr().String())
		conn.Close()
		recordRuntimeError(err)
		return
	}
	if sec != nil && strings.TrimSpace(sec.RevocationCacheFile) != "" {
		revoked, revErr := globalRevocationCache.IsRevoked(sec.RevocationCacheFile, clientPubKey)
		if revErr != nil {
			log.Printf("Warning: revocation cache check failed: %v", revErr)
			recordRuntimeError(revErr)
		} else if revoked {
			log.Printf("Connection from %s rejected: client certificate is revoked", conn.RemoteAddr())
			recordEvent("provider", "reject_revoked", conn.RemoteAddr().String())
			conn.Close()
			recordRuntimeError(fmt.Errorf("revoked client certificate: %s", hex.EncodeToString(clientPubKey.SerializeCompressed())))
			return
		}
	}
	if !authManager.IsPeerAuthorized(clientPubKey) {
		log.Printf("Connection from %s rejected: client %s is not authorized", conn.RemoteAddr(), hex.EncodeToString(clientPubKey.SerializeCompressed()))
		recordEvent("provider", "reject_unauthorized", conn.RemoteAddr().String())
		conn.Close()
		recordRuntimeError(fmt.Errorf("unauthorized client: %s", hex.EncodeToString(clientPubKey.SerializeCompressed())))
		return
	}

	log.Printf("Accepted connection from authorized client %s (%s)", hex.EncodeToString(clientPubKey.SerializeCompressed()), conn.RemoteAddr())
	recordEvent("provider", "client_connected", conn.RemoteAddr().String())

	// Allocate Dynamic IP
	assignedIP, err := ipPool.Allocate()
	if err != nil {
		log.Printf("Failed to allocate IP for client: %v", err)
		conn.Close()
		recordRuntimeError(err)
		return
	}

	// Handshake: Send assigned IP to client
	if _, err := conn.Write([]byte(assignedIP.String() + "\n")); err != nil {
		log.Printf("Failed to send IP assignment: %v", err)
		ipPool.Release(assignedIP)
		conn.Close()
		recordRuntimeError(err)
		return
	}

	session := newClientSession(conn, limitBytesPerSec)
	if !clients.AddSession(assignedIP, session, cfg.MaxConsumers) {
		log.Printf("Connection from %s rejected: provider at max consumer capacity (%d)", conn.RemoteAddr(), cfg.MaxConsumers)
		recordEvent("provider", "reject_capacity", conn.RemoteAddr().String())
		recordRuntimeError(fmt.Errorf("provider max consumer capacity reached"))
		ipPool.Release(assignedIP)
		conn.Close()
		return
	}

	// Handle client traffic and session
	go func(c net.Conn, ip net.IP, pk *btcec.PublicKey) {
		defer c.Close()
		defer ipPool.Release(ip)
		defer func() {
			clients.RemoveSession(ip)
			log.Printf("Client %s (%s) disconnected.", hex.EncodeToString(pk.SerializeCompressed()), c.RemoteAddr())
			recordEvent("provider", "client_disconnected", c.RemoteAddr().String())
		}()

		expiration, ok := authManager.GetPeerExpiration(pk)
		if !ok {
			log.Printf("Logic error: authorized client %s has no expiration. Disconnecting.", hex.EncodeToString(pk.SerializeCompressed()))
			return
		}

		// Cap session duration by provider limit if configured.
		if cfg.MaxSessionDurationSecs > 0 {
			maxExpiry := time.Now().Add(time.Duration(cfg.MaxSessionDurationSecs) * time.Second)
			if maxExpiry.Before(expiration) {
				expiration = maxExpiry
				log.Printf("Session for client %s capped to %ds by provider policy", hex.EncodeToString(pk.SerializeCompressed()), cfg.MaxSessionDurationSecs)
			}
		}

		sessionTimer := time.NewTimer(time.Until(expiration))
		done := make(chan struct{})

		go func(sess *clientSession) {
			copyStreamWithControl(iface, c, sess.stats.addUpstream, sess.upstreamLimiter)
			close(done)
		}(session)

		select {
		case <-sessionTimer.C:
			log.Printf("Session expired for client %s. Disconnecting.", hex.EncodeToString(pk.SerializeCompressed()))
		case <-done:
			sessionTimer.Stop()
		}
		if !sessionTimer.Stop() {
			<-sessionTimer.C
		}
		log.Printf(
			"Session stats client=%s duration=%s upstream_bytes=%d downstream_bytes=%d",
			hex.EncodeToString(pk.SerializeCompressed()),
			time.Since(session.stats.startedAt).Round(time.Second),
			session.stats.upstreamBytes.Load(),
			session.stats.downstreamBytes.Load(),
		)
		recordTraffic(session.stats.upstreamBytes.Load(), session.stats.downstreamBytes.Load())
	}(conn, assignedIP, clientPubKey)
}

func readTunLoop(iface *water.Interface, clients *ClientMap) {
	// 65535 bytes covers the maximum theoretical IP packet size (full 16-bit length field).
	packet := make([]byte, 65535)
	for {
		n, err := iface.Read(packet)
		if err != nil {
			// This error is expected when the interface is closed.
			return
		}

		// Check minimum packet size for IP header
		if n < 1 {
			continue
		}

		version := packet[0] >> 4
		if version == 4 {
			// IPv4 Header: Version (byte 0 >> 4) should be 4. Dest IP is bytes 12-16 (offset 12-15 in 0-indexed).
			// IP header length is at lower 4 bits of byte 0, multiplied by 4 gives header length.
			if n < 20 {
				log.Printf("Info: readTunLoop dropping truncated IPv4 packet (len=%d)", n)
				continue
			}
			headerLen := int(packet[0]&0x0F) * 4
			if n < headerLen {
				log.Printf("Info: readTunLoop dropping truncated IPv4 packet (len=%d, header=%d)", n, headerLen)
				continue
			}
			destIP := net.IP(packet[16:20])
			clients.mu.RLock()
			session, ok := clients.m[destIP.String()]
			clients.mu.RUnlock()
			if ok {
				if session.downLimiter != nil {
					session.downLimiter.accountAndThrottle(n)
				}
				session.stats.addDownstream(n)
				if _, err := session.conn.Write(packet[:n]); err != nil {
					log.Printf("Info: failed to write packet to client %s: %v", destIP.String(), err)
				}
			}
		} else if version == 6 {
			// IPv6 Header: Version (byte 0 >> 4) should be 6. Dest IP is bytes 24-40.
			if n < 40 {
				log.Printf("Info: readTunLoop dropping truncated IPv6 packet (len=%d)", n)
				continue
			}
			destIP := net.IP(packet[24:40])
			clients.mu.RLock()
			session, ok := clients.m[destIP.String()]
			clients.mu.RUnlock()
			if ok {
				if session.downLimiter != nil {
					session.downLimiter.accountAndThrottle(n)
				}
				session.stats.addDownstream(n)
				if _, err := session.conn.Write(packet[:n]); err != nil {
					log.Printf("Info: failed to write IPv6 packet to client %s: %v", destIP.String(), err)
				}
			}
		} else {
			log.Printf("Info: readTunLoop dropping unknown IP version packet (version=%d, len=%d)", version, n)
		}
	}
}

// ConnectToProvider connects to a provider and sets up the client-side tunnel.
// If spendingMgr is provided, it will be used for spending limit checks and auto-recharge.
func ConnectToProvider(ctx context.Context, cfg *config.ClientConfig, sec *config.SecurityConfig, localPrivKey *btcec.PrivateKey, serverPubKey *btcec.PublicKey, endpoint string, expected ClientSecurityExpectations, spendingMgr *SpendingManager) error {
	if err := EnsureElevatedPrivileges(); err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("client requires automatic networking privileges: %w", err)
	}
	metricsToken := ""
	if sec != nil {
		metricsToken = strings.TrimSpace(sec.MetricsAuthToken)
	}
	startMetricsServer(cfg.MetricsListenAddr, metricsToken)
	setClientConnected(true)
	defer setClientConnected(false)

	tlsPolicy := TLSPolicy{}
	var err error
	var customCiphers []string
	if sec != nil {
		customCiphers = sec.TlsCustomCipherSuites
	}
	if sec != nil {
		tlsPolicy, err = ResolveTLSPolicy(sec.TLSMinVersion, sec.TLSProfile, customCiphers)
		if err != nil {
			return fmt.Errorf("failed to resolve TLS policy: %w", err)
		}
	} else {
		tlsPolicy, _ = ResolveTLSPolicy("", "", nil)
	}

	if sec != nil && strings.TrimSpace(sec.RevocationCacheFile) != "" {
		revoked, revErr := globalRevocationCache.IsRevoked(sec.RevocationCacheFile, serverPubKey)
		if revErr != nil {
			recordRuntimeError(revErr)
			log.Printf("Warning: revocation cache check failed: %v", revErr)
		} else if revoked {
			err := fmt.Errorf("provider certificate is revoked")
			recordRuntimeError(err)
			return err
		}
	}

	tlsConfig, err := GenerateClientTLSConfig(localPrivKey, serverPubKey, tlsPolicy)
	if err != nil {
		return fmt.Errorf("failed to generate client TLS config: %w", err)
	}

	providerHost, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint format: %w", err)
	}
	preConnectIP, preConnectErr := util.GetPublicIP()
	if preConnectErr != nil {
		log.Printf("Warning: could not determine pre-connect public IP: %v", preConnectErr)
	} else {
		log.Printf("Pre-connect public IP: %s", preConnectIP.String())
	}

	log.Printf("Dialing %s...", endpoint)
	recordEvent("client", "connect_attempt", endpoint)

	conn, err := transport.Dial(ctx, endpoint, tlsConfig, false)
	if err != nil {
		if cfg.EnableWebSocketFallback {
			log.Printf("Direct connect failed, trying WebSocket fallback...")
			conn, err = transport.Dial(ctx, endpoint, tlsConfig, true)
		}
		if err != nil {
			recordRuntimeError(err)
			recordEvent("client", "connect_failed", endpoint)
			return fmt.Errorf("failed to connect to provider: %w", err)
		}
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// Handshake: Read assigned IP
	ipStr, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("failed to read assigned IP from provider: %w", err)
	}
	assignedIP := strings.TrimSpace(ipStr)
	log.Printf("Provider assigned IP: %s", assignedIP)

	iface, err := createTunInterface(cfg.InterfaceName, assignedIP, cfg.TunSubnet)
	if err != nil {
		recordRuntimeError(err)
		return fmt.Errorf("failed to create client TUN interface: %w", err)
	}
	defer iface.Close()

	cleanupNetwork, err := configureClientNetwork(iface.Name(), providerHost)
	if err != nil {
		log.Printf("Warning: automatic route/DNS setup unavailable: %v", err)
	} else if cleanupNetwork != nil {
		defer cleanupNetwork()
	}

	if cfg.EnableKillSwitch {
		cleanupKillSwitch, err := setupKillSwitch(iface.Name(), providerHost)
		if err != nil {
			recordRuntimeError(err)
			return fmt.Errorf("failed to activate kill switch: %w", err)
		}
		if cleanupKillSwitch != nil {
			defer cleanupKillSwitch()
		}
		log.Printf("Kill switch enabled for session on interface %s", iface.Name())
	}

	expected.ProviderHost = providerHost
	expected.StrictVerification = cfg.StrictVerification
	expected.VerifyThroughputAfter = cfg.VerifyThroughputAfterSetup
	checkCtx, cancelChecks := context.WithTimeout(context.Background(), 20*time.Second)
	if err := runClientPostConnectChecks(checkCtx, expected, preConnectIP); err != nil {
		cancelChecks()
		recordRuntimeError(err)
		return err
	}
	cancelChecks()

	log.Printf("Successfully connected to %s. Tunnel is active.", endpoint)
	recordEvent("client", "connected", endpoint)

	// Forward packets
	go copyStreamWithControl(conn, iface, func(n int) { recordTraffic(int64(n), 0) }, nil)
	copyStreamWithControl(iface, conn, func(n int) { recordTraffic(0, int64(n)) }, nil) // Block on this one until connection closes
	recordEvent("client", "disconnected", endpoint)

	return nil
}
