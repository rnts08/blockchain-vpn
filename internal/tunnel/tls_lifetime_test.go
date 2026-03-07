package tunnel

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestGenerateSelfSignedCertLifetime(t *testing.T) {
	key, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	const lifetime = 2 * time.Hour
	cert, err := generateSelfSignedCert(key, lifetime)
	if err != nil {
		t.Fatalf("generateSelfSignedCert failed: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("certificate chain is empty")
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse cert failed: %v", err)
	}

	gotLifetime := parsed.NotAfter.Sub(parsed.NotBefore)
	if gotLifetime < lifetime-2*time.Minute || gotLifetime > lifetime+2*time.Minute {
		t.Fatalf("unexpected cert lifetime: got %s want around %s", gotLifetime, lifetime)
	}
}
