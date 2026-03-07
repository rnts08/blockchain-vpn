package tunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"blockchain-vpn/internal/auth"
	"blockchain-vpn/internal/config"

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

// copyStream copies data between two ReadWriteClosers and logs errors.
func copyStream(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		// This error is expected when a connection is closed, so we log it lightly.
		log.Printf("Info: stream copy ended: %v", err)
	}
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

// Allocate finds the next available IP in the /24 subnet.
func (p *IPPool) Allocate() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ip := make(net.IP, len(p.baseIP))
	copy(ip, p.baseIP)
	ip = ip.To4()
	if ip == nil {
		return nil, fmt.Errorf("only IPv4 supported for simple pool")
	}

	// Simple allocation strategy: iterate 2 to 254 (assuming .1 is gateway)
	// This assumes a /24 subnet.
	for i := 2; i < 255; i++ {
		ip[3] = byte(i)
		ipStr := ip.String()
		if ipStr == p.baseIP.String() {
			continue
		}
		if !p.used[ipStr] {
			p.used[ipStr] = true
			return ip, nil
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
	m  map[string]net.Conn
	mu sync.RWMutex
}

// StartProviderServer sets up the TUN interface and listens for TLS connections.
func StartProviderServer(ctx context.Context, cfg *config.ProviderConfig, privKey *btcec.PrivateKey, authManager *auth.AuthManager) {
	tlsConfig, err := GenerateServerTLSConfig(privKey)
	if err != nil {
		log.Fatalf("Failed to generate server TLS config: %v", err)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", cfg.ListenPort)
	listener, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to start TLS listener: %v", err)
	}
	defer listener.Close()
	log.Printf("Provider server listening on %s", listenAddr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	iface, err := createTunInterface(cfg.InterfaceName, cfg.TunIP, cfg.TunSubnet)
	if err != nil {
		log.Fatalf("Failed to create provider TUN interface: %v", err)
	}
	defer iface.Close()

	// Initialize IP Pool and Client Map
	ipPool := NewIPPool(cfg.TunIP)
	clients := &ClientMap{m: make(map[string]net.Conn)}

	// Start the central TUN reader goroutine
	// This reads packets from the TUN interface and routes them to the correct client connection.
	go readTunLoop(iface, clients)

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				log.Println("Provider server shutting down.")
				return
			default:
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}

		// The handshake is already complete from Accept().
		// Now we verify the peer's certificate against our authorization list.
		state := conn.(*tls.Conn).ConnectionState()
		if len(state.PeerCertificates) == 0 {
			log.Printf("Connection from %s rejected: no client certificate provided", conn.RemoteAddr())
			conn.Close()
			continue
		}

		clientPubKey, err := certToBTCECPubKey(state.PeerCertificates[0])
		if err != nil {
			log.Printf("Connection from %s rejected: %v", conn.RemoteAddr(), err)
			conn.Close()
			continue
		}
		if !authManager.IsPeerAuthorized(clientPubKey) {
			log.Printf("Connection from %s rejected: client %s is not authorized", conn.RemoteAddr(), hex.EncodeToString(clientPubKey.SerializeCompressed()))
			conn.Close()
			continue
		}

		log.Printf("Accepted connection from authorized client %s (%s)", hex.EncodeToString(clientPubKey.SerializeCompressed()), conn.RemoteAddr())

		// Allocate Dynamic IP
		assignedIP, err := ipPool.Allocate()
		if err != nil {
			log.Printf("Failed to allocate IP for client: %v", err)
			conn.Close()
			continue
		}

		// Handshake: Send assigned IP to client
		if _, err := conn.Write([]byte(assignedIP.String() + "\n")); err != nil {
			log.Printf("Failed to send IP assignment: %v", err)
			ipPool.Release(assignedIP)
			conn.Close()
			continue
		}

		// Register client
		clients.mu.Lock()
		clients.m[assignedIP.String()] = conn
		clients.mu.Unlock()

		// Handle client traffic and session
		go func(c net.Conn, ip net.IP, pk *btcec.PublicKey) {
			defer c.Close()
			defer ipPool.Release(ip)
			defer func() {
				clients.mu.Lock()
				delete(clients.m, ip.String())
				clients.mu.Unlock()
				log.Printf("Client %s (%s) disconnected.", hex.EncodeToString(pk.SerializeCompressed()), c.RemoteAddr())
			}()

			expiration, ok := authManager.GetPeerExpiration(pk)
			if !ok {
				log.Printf("Logic error: authorized client %s has no expiration. Disconnecting.", hex.EncodeToString(pk.SerializeCompressed()))
				return
			}

			sessionTimer := time.NewTimer(time.Until(expiration))
			done := make(chan struct{})

			go func() {
				copyStream(iface, c)
				close(done)
			}()

			select {
			case <-sessionTimer.C:
				log.Printf("Session expired for client %s. Disconnecting.", hex.EncodeToString(pk.SerializeCompressed()))
			case <-done:
				sessionTimer.Stop() // Connection closed by client, clean up timer.
			}
		}(conn, assignedIP, clientPubKey)
	}
}

func readTunLoop(iface *water.Interface, clients *ClientMap) {
	packet := make([]byte, 2048)
	for {
		n, err := iface.Read(packet)
		if err != nil {
			// This error is expected when the interface is closed.
			return
		}

		// Parse Destination IP (IPv4)
		// IPv4 Header: Version (byte 0 >> 4) should be 4. Dest IP is bytes 16-20.
		if n >= 20 && (packet[0]>>4) == 4 {
			destIP := net.IP(packet[16:20])
			clients.mu.RLock()
			conn, ok := clients.m[destIP.String()]
			clients.mu.RUnlock()
			if ok {
				conn.Write(packet[:n])
			}
		}
	}
}

// ConnectToProvider connects to a provider and sets up the client-side tunnel.
func ConnectToProvider(ctx context.Context, cfg *config.ClientConfig, localPrivKey *btcec.PrivateKey, serverPubKey *btcec.PublicKey, endpoint string) error {
	tlsConfig, err := GenerateClientTLSConfig(localPrivKey, serverPubKey)
	if err != nil {
		return fmt.Errorf("failed to generate client TLS config: %w", err)
	}

	providerHost, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("invalid endpoint format: %w", err)
	}

	log.Printf("Dialing %s...", endpoint)
	conn, err := tls.Dial("tcp", endpoint, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to provider: %w", err)
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	// Handshake: Read assigned IP
	ipStr, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read assigned IP from provider: %w", err)
	}
	assignedIP := strings.TrimSpace(ipStr)
	log.Printf("Provider assigned IP: %s", assignedIP)

	iface, err := createTunInterface(cfg.InterfaceName, assignedIP, cfg.TunSubnet)
	if err != nil {
		return fmt.Errorf("failed to create client TUN interface: %w", err)
	}
	defer iface.Close()

	cleanupNetwork, err := configureClientNetwork(iface.Name(), providerHost)
	if err != nil {
		log.Printf("Warning: automatic route/DNS setup unavailable: %v", err)
	} else if cleanupNetwork != nil {
		defer cleanupNetwork()
	}

	log.Printf("Successfully connected to %s. Tunnel is active.", endpoint)

	// Forward packets
	go copyStream(conn, iface)
	copyStream(iface, conn) // Block on this one until connection closes

	return nil
}
