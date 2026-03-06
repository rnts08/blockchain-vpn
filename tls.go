package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

// GenerateTLSConfig creates a TLS configuration from a private key.
// For servers, it creates a config that requires client authentication.
// For clients, it creates a config that verifies the server.
func GenerateTLSConfig(privKey *btcec.PrivateKey, rootCAs *x509.CertPool, isServer bool) (*tls.Config, error) {
	// Generate a self-signed certificate from the private key
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"BlockchainVPN"},
			CommonName:   "p2p-node-" + privKey.PubKey().String(),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year validity
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privKey.PubKey().ToECDSA(), privKey.ToECDSA())
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  (*ecdsa.PrivateKey)(privKey.ToECDSA()),
	}

	if isServer {
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAnyClientCert, // We will verify the cert manually against the AuthManager
			MinVersion:   tls.VersionTLS13,
		}, nil
	}

	// Client config
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCAs, // Pool with server's cert
		ServerName:   "p2p-node-" + privKey.PubKey().String(), // For SNI
		MinVersion:   tls.VersionTLS13,
	}, nil
}