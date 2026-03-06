package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"math/big"
	"os/exec"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/songgao/water"
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
		if err := exec.Command("ip", "addr", "add", ip+"/"+subnetMask, "dev", iface.Name()).Run(); err != nil {
			return nil, fmt.Errorf("failed to assign IP to TUN interface: %w", err)
		}
		if err := exec.Command("ip", "link", "set", "up", "dev", iface.Name()).Run(); err != nil {
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

// StartProviderServer sets up the TUN interface and listens for TLS connections.
func StartProviderServer(cfg *ProviderConfig, privKey *btcec.PrivateKey, authManager *AuthManager) {
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

	iface, err := createTunInterface(cfg.InterfaceName, cfg.TunIP, cfg.TunSubnet)
	if err != nil {
		log.Fatalf("Failed to create provider TUN interface: %v", err)
	}
	defer iface.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
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
		// Forward packets between the TUN interface and the TLS connection
		go copyStream(conn, iface)
		go copyStream(iface, conn)
	}
}

// ConnectToProvider connects to a provider and sets up the client-side tunnel.
func ConnectToProvider(cfg *ClientConfig, localPrivKey *btcec.PrivateKey, serverPubKey *btcec.PublicKey, endpoint string) error {
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

	iface, err := createTunInterface(cfg.InterfaceName, cfg.TunIP, cfg.TunSubnet)
	if err != nil {
		return fmt.Errorf("failed to create client TUN interface: %w", err)
	}
	defer iface.Close()

	log.Printf("Dialing %s...", endpoint)
	conn, err := tls.Dial("tcp", endpoint, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to provider: %w", err)
	}
	defer conn.Close()
	log.Printf("Successfully connected to %s. Tunnel is active.", endpoint)

	// Forward packets
	go copyStream(conn, iface)
	copyStream(iface, conn) // Block on this one until connection closes

	return nil
}