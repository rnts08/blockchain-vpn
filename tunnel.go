package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

// createTunInterface creates and configures a new TUN interface.
func createTunInterface(ifaceName string, ip string, subnetMask string) (*water.Interface, error) {
	iface, err := water.New(water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: ifaceName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TUN interface: %w", err)
	}

	log.Printf("TUN interface %s created", iface.Name())

	// OS-specific IP configuration
	if runtime.GOOS == "linux" {
		link, err := netlink.LinkByName(iface.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to get link for %s: %w", iface.Name(), err)
		}
		addr, err := netlink.ParseAddr(ip + "/" + subnetMask)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address: %w", err)
		}
		if err := netlink.AddrAdd(link, addr); err != nil {
			return nil, fmt.Errorf("failed to assign IP to TUN interface: %w", err)
		}
		if err := netlink.LinkSetUp(link); err != nil {
			return nil, fmt.Errorf("failed to bring up TUN interface: %w", err)
		}
	} else {
		log.Printf("Warning: automatic IP configuration for TUN interfaces is only supported on Linux. Please configure %s with IP %s manually.", iface.Name(), ip)
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
func StartProviderServer(ctx context.Context, cfg *ProviderConfig, privKey *btcec.PrivateKey, authManager *AuthManager) {
	caPool := x509.NewCertPool() // We don't use a CA pool, we verify keys directly.

	tlsConfig, err := GenerateTLSConfig(privKey, caPool, true)
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

		clientCert := state.PeerCertificates[0]
		ecdsaPubKey, ok := clientCert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			log.Printf("Connection from %s rejected: client cert public key is not ECDSA", conn.RemoteAddr())
			conn.Close()
			continue
		}

		clientPubKey := btcec.NewPublicKeyFromECDSA(ecdsaPubKey)
		if !authManager.IsPeerAuthorized(clientPubKey) {
			log.Printf("Connection from %s rejected: client %s is not authorized", conn.RemoteAddr(), clientPubKey.String())
			conn.Close()
			continue
		}

		log.Printf("Accepted connection from authorized client %s (%s)", clientPubKey.String(), conn.RemoteAddr())

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
				log.Printf("Client %s (%s) disconnected.", pk.String(), c.RemoteAddr())
			}()

			expiration, ok := authManager.GetPeerExpiration(pk)
			if !ok {
				log.Printf("Logic error: authorized client %s has no expiration. Disconnecting.", pk.String())
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
				log.Printf("Session expired for client %s. Disconnecting.", pk.String())
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
func ConnectToProvider(ctx context.Context, cfg *ClientConfig, localPrivKey *btcec.PrivateKey, serverPubKey *btcec.PublicKey, endpoint string) error {
	// Create a CA pool with the server's public key.
	// This is a simplified trust model where we trust the public key from the blockchain.
	// We generate a placeholder certificate for the server's public key to satisfy the crypto/tls library.
	serverCertTemplate := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
	}
	serverCertBytes, err := x509.CreateCertificate(rand.Reader, &serverCertTemplate, &serverCertTemplate, serverPubKey.ToECDSA(), localPrivKey.ToECDSA())
	if err != nil {
		return fmt.Errorf("failed to create placeholder server cert: %w", err)
	}
	serverCert, err := x509.ParseCertificate(serverCertBytes)
	if err != nil {
		return fmt.Errorf("failed to parse placeholder server cert: %w", err)
	}
	caPool := x509.NewCertPool()
	caPool.AddCert(serverCert)

	tlsConfig, err := GenerateTLSConfig(localPrivKey, caPool, false)
	if err != nil {
		return fmt.Errorf("failed to generate client TLS config: %w", err)
	}

	// Before dialing, get original route info
	var defaultGW string
	if runtime.GOOS == "linux" {
		var err error
		defaultGW, err = getDefaultGateway()
		if err != nil {
			log.Printf("Warning: could not get default gateway, will not be able to route all traffic through tunnel: %v", err)
		}
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

	// Configure Routing and DNS
	if runtime.GOOS == "linux" && defaultGW != "" {
		if err := setupRouting(iface.Name(), providerHost, defaultGW); err != nil {
			return err
		}
		defer restoreRouting(iface.Name(), providerHost)

		if err := setupDNS(); err != nil {
			log.Printf("Warning: failed to setup DNS: %v", err)
		} else {
			defer restoreDNS()
		}
	}

	log.Printf("Successfully connected to %s. Tunnel is active.", endpoint)

	// Forward packets
	go copyStream(conn, iface)
	copyStream(iface, conn) // Block on this one until connection closes

	return nil
}

func setupRouting(ifaceName, providerHost, defaultGW string) error {
	log.Println("Modifying routing table for full tunnel...")
	link, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return fmt.Errorf("could not get interface %s: %w", ifaceName, err)
	}

	// 1. Add route for provider to go through original gateway to avoid routing loops
	gw := net.ParseIP(defaultGW)
	providerIP := net.ParseIP(providerHost)
	if gw == nil || providerIP == nil {
		return fmt.Errorf("invalid IP address for gateway or provider")
	}

	routeToProvider := &netlink.Route{
		Dst: &net.IPNet{IP: providerIP, Mask: net.CIDRMask(32, 32)},
		Gw:  gw,
	}
	if err := netlink.RouteAdd(routeToProvider); err != nil {
		log.Printf("Warning: failed to add route for vpn provider endpoint: %v", err)
	}

	// 2. Change default route to go through TUN by adding two more-specific routes
	_, dst1, _ := net.ParseCIDR("0.0.0.0/1")
	route1 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst1,
	}
	if err := netlink.RouteAdd(route1); err != nil {
		return fmt.Errorf("failed to set default route part 1: %w", err)
	}

	_, dst2, _ := net.ParseCIDR("128.0.0.0/1")
	route2 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst2,
	}
	if err := netlink.RouteAdd(route2); err != nil {
		netlink.RouteDel(route1) // Cleanup part 1 before failing
		return fmt.Errorf("failed to set default route part 2: %w", err)
	}

	return nil
}

func restoreRouting(ifaceName, providerHost string) {
	log.Println("Restoring original routing table...")
	// The routes are often deleted automatically when the interface goes down,
	// but we attempt to clean them up explicitly just in case.
	// Errors are ignored here as the routes may already be gone.
	providerIP := net.ParseIP(providerHost)
	netlink.RouteDel(&netlink.Route{Dst: &net.IPNet{IP: providerIP, Mask: net.CIDRMask(32, 32)}})
	_, dst1, _ := net.ParseCIDR("0.0.0.0/1")
	_, dst2, _ := net.ParseCIDR("128.0.0.0/1")
	link, err := netlink.LinkByName(ifaceName)
	if err == nil {
		netlink.RouteDel(&netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst1})
		netlink.RouteDel(&netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst2})
	}
}

const resolvConfPath = "/etc/resolv.conf"
const resolvBackupPath = "/etc/resolv.conf.bcvpn-backup"
const secureDNS = "nameserver 1.1.1.1\nnameserver 8.8.8.8\n"

func setupDNS() error {
	log.Println("Configuring secure DNS...")
	// 1. Backup existing resolv.conf
	content, err := os.ReadFile(resolvConfPath)
	if err != nil {
		return fmt.Errorf("failed to read resolv.conf: %w", err)
	}
	if err := os.WriteFile(resolvBackupPath, content, 0644); err != nil {
		return fmt.Errorf("failed to backup resolv.conf: %w", err)
	}

	// 2. Write new resolv.conf
	// Note: This is a simple overwrite. On modern Linux with systemd-resolved,
	// this might be overwritten by the system or require using resolvectl.
	// For this implementation, we assume a simple environment or that overwriting works temporarily.
	if err := os.WriteFile(resolvConfPath, []byte(secureDNS), 0644); err != nil {
		return fmt.Errorf("failed to write new resolv.conf: %w", err)
	}
	return nil
}

func restoreDNS() {
	log.Println("Restoring DNS settings...")
	content, err := os.ReadFile(resolvBackupPath)
	if err != nil {
		log.Printf("Error reading DNS backup: %v", err)
		return
	}
	if err := os.WriteFile(resolvConfPath, content, 0644); err != nil {
		log.Printf("Error restoring DNS: %v", err)
	}
	os.Remove(resolvBackupPath)
}
