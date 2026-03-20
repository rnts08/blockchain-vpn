package tunnel

import (
	"context"
	"crypto/tls"
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

func TestBuildRotatingServerTLSConfig(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := buildRotatingServerTLSConfig(ctx, priv, time.Hour, time.Minute, TLSPolicy{})
	if err != nil {
		t.Fatalf("buildRotatingServerTLSConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil tls.Config")
	}
	if cfg.ClientAuth != tls.RequireAnyClientCert {
		t.Errorf("expected RequireAnyClientCert, got %v", cfg.ClientAuth)
	}
	if cfg.GetCertificate == nil {
		t.Fatal("expected GetCertificate to be set")
	}

	cert, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if cert == nil {
		t.Fatal("expected non-nil certificate from GetCertificate")
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("expected certificate chain")
	}
}

func TestBuildRotatingServerTLSConfigDefaultValues(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := buildRotatingServerTLSConfig(ctx, priv, 0, 0, TLSPolicy{})
	if err != nil {
		t.Fatalf("buildRotatingServerTLSConfig failed: %v", err)
	}

	if cfg.MinVersion != tls.VersionTLS13 {
		t.Errorf("expected default MinVersion TLS 1.3, got 0x%x", cfg.MinVersion)
	}

	cert, err := cfg.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse cert failed: %v", err)
	}
	expectedLifetime := defaultCertLifetime
	gotLifetime := parsed.NotAfter.Sub(parsed.NotBefore)
	if gotLifetime < expectedLifetime-2*time.Minute || gotLifetime > expectedLifetime+2*time.Minute {
		t.Errorf("unexpected default cert lifetime: got %s want around %s", gotLifetime, expectedLifetime)
	}
}

func TestBuildRotatingServerTLSConfigNoPanicAfterCancel(t *testing.T) {
	t.Parallel()
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg, err := buildRotatingServerTLSConfig(ctx, priv, time.Hour, time.Minute, TLSPolicy{})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = cfg.GetCertificate(nil)
	if err != nil {
		t.Fatalf("GetCertificate should not fail after context cancel: %v", err)
	}
}
