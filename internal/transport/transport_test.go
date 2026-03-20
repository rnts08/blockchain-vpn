package transport

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
)

func TestWSConnImplementsNetConn(t *testing.T) {
	t.Parallel()
	var _ net.Conn = &WSConn{}
}

func TestDialInvalidAddress(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	ctx := context.Background()

	_, err := Dial(ctx, "invalid:address:99999", tlsConfig, false)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

func TestDialContextCancelled(t *testing.T) {
	t.Parallel()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Dial(ctx, "127.0.0.1:1", tlsConfig, false)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
