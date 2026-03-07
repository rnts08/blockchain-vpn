package tunnel

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

func buildRotatingServerTLSConfig(ctx context.Context, privKey *btcec.PrivateKey, lifetime, rotateBefore time.Duration) (*tls.Config, error) {
	if lifetime <= 0 {
		lifetime = defaultCertLifetime
	}
	if rotateBefore <= 0 || rotateBefore >= lifetime {
		rotateBefore = lifetime / 10
	}

	current := &atomic.Pointer[tls.Certificate]{}
	cert, err := generateSelfSignedCert(privKey, lifetime)
	if err != nil {
		return nil, err
	}
	current.Store(&cert)

	rotateEvery := lifetime - rotateBefore
	go func() {
		ticker := time.NewTicker(rotateEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				next, err := generateSelfSignedCert(privKey, lifetime)
				if err != nil {
					log.Printf("Warning: failed to rotate TLS certificate: %v", err)
					continue
				}
				current.Store(&next)
				log.Printf("Rotated provider TLS certificate (lifetime=%s)", lifetime)
			}
		}
	}()

	return &tls.Config{
		ClientAuth: tls.RequireAnyClientCert,
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			c := current.Load()
			if c == nil {
				return nil, fmt.Errorf("no active server certificate")
			}
			return c, nil
		},
	}, nil
}
