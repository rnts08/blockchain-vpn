package tunnel

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestTLSHandshakeWithOnChainIdentity(t *testing.T) {
	serverKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	clientKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	policy, err := ResolveTLSPolicy("", "")
	if err != nil {
		t.Fatalf("resolve tls policy: %v", err)
	}
	serverTLS, err := GenerateServerTLSConfig(serverKey, time.Hour, policy)
	if err != nil {
		t.Fatalf("server TLS config: %v", err)
	}
	clientTLS, err := GenerateClientTLSConfig(clientKey, serverKey.PubKey(), policy)
	if err != nil {
		t.Fatalf("client TLS config: %v", err)
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	_ = c1.SetDeadline(time.Now().Add(5 * time.Second))
	_ = c2.SetDeadline(time.Now().Add(5 * time.Second))

	srv := tls.Server(c1, serverTLS)
	cli := tls.Client(c2, clientTLS)

	errCh := make(chan error, 2)
	go func() { errCh <- srv.Handshake() }()
	go func() { errCh <- cli.Handshake() }()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("handshake failed: %v", err)
		}
	}
}

func TestTLSHandshakeRejectsWrongServerIdentity(t *testing.T) {
	serverKey, _ := btcec.NewPrivateKey()
	clientKey, _ := btcec.NewPrivateKey()
	wrongExpectedKey, _ := btcec.NewPrivateKey()

	policy, err := ResolveTLSPolicy("", "")
	if err != nil {
		t.Fatalf("resolve tls policy: %v", err)
	}
	serverTLS, err := GenerateServerTLSConfig(serverKey, time.Hour, policy)
	if err != nil {
		t.Fatalf("server TLS config: %v", err)
	}
	clientTLS, err := GenerateClientTLSConfig(clientKey, wrongExpectedKey.PubKey(), policy)
	if err != nil {
		t.Fatalf("client TLS config: %v", err)
	}

	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()
	_ = c1.SetDeadline(time.Now().Add(5 * time.Second))
	_ = c2.SetDeadline(time.Now().Add(5 * time.Second))

	srv := tls.Server(c1, serverTLS)
	cli := tls.Client(c2, clientTLS)

	errCh := make(chan error, 2)
	go func() { errCh <- srv.Handshake() }()
	go func() { errCh <- cli.Handshake() }()

	var gotErr bool
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Fatal("expected handshake failure with wrong server identity, got success")
	}
}
