package tunnel

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

var identityExtensionOID = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 55555, 1, 1}

const defaultCertLifetime = 365 * 24 * time.Hour

func generateSelfSignedCert(privKey *btcec.PrivateKey, lifetime time.Duration) (tls.Certificate, error) {
	pubKeyID := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	tlsPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to generate TLS keypair: %w", err)
	}
	if lifetime <= 0 {
		lifetime = defaultCertLifetime
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"BlockchainVPN"},
			CommonName:   "p2p-node-" + pubKeyID,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(lifetime),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		ExtraExtensions: []pkix.Extension{
			{
				Id:    identityExtensionOID,
				Value: privKey.PubKey().SerializeCompressed(),
			},
		},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &tlsPrivKey.PublicKey, tlsPrivKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  tlsPrivKey,
	}, nil
}

func certToBTCECPubKey(cert *x509.Certificate) (*btcec.PublicKey, error) {
	for _, ext := range cert.Extensions {
		if ext.Id.Equal(identityExtensionOID) {
			key, err := btcec.ParsePubKey(ext.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid identity extension public key: %w", err)
			}
			return key, nil
		}
	}

	ecdsaPubKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("peer certificate public key is not ECDSA")
	}
	// Backward compatibility path for legacy certificates where the TLS cert key
	// was expected to be the same secp256k1 key.
	compressed := elliptic.MarshalCompressed(ecdsaPubKey.Curve, ecdsaPubKey.X, ecdsaPubKey.Y)
	return btcec.ParsePubKey(compressed)
}

func GenerateServerTLSConfig(privKey *btcec.PrivateKey, lifetime time.Duration) (*tls.Config, error) {
	cert, err := generateSelfSignedCert(privKey, lifetime)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

func GenerateClientTLSConfig(privKey *btcec.PrivateKey, expectedServerKey *btcec.PublicKey) (*tls.Config, error) {
	cert, err := generateSelfSignedCert(privKey, defaultCertLifetime)
	if err != nil {
		return nil, err
	}
	if expectedServerKey == nil {
		return nil, fmt.Errorf("expected server public key is required")
	}

	expected := expectedServerKey.SerializeCompressed()

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return fmt.Errorf("server did not provide a certificate")
			}
			cert, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse server certificate: %w", err)
			}
			peerKey, err := certToBTCECPubKey(cert)
			if err != nil {
				return err
			}
			if !peerKey.IsEqual(expectedServerKey) {
				return fmt.Errorf("server identity mismatch: expected %x, got %x", expected, peerKey.SerializeCompressed())
			}
			return nil
		},
	}, nil
}
