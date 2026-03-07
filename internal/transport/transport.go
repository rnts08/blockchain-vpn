package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/btcsuite/websocket"
)

// WSConn wraps a websocket.Conn to implement the net.Conn interface.
type WSConn struct {
	*websocket.Conn
	reader io.Reader
}

// Read implements the net.Conn Read method.
func (c *WSConn) Read(b []byte) (int, error) {
	for {
		if c.reader == nil {
			_, r, err := c.NextReader()
			if err != nil {
				return 0, err
			}
			c.reader = r
		}
		n, err := c.reader.Read(b)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue // Look for the next message
		}
		return n, err
	}
}

// Write implements the net.Conn Write method.
func (c *WSConn) Write(b []byte) (int, error) {
	err := c.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// LocalAddr implements the net.Conn LocalAddr method.
func (c *WSConn) LocalAddr() net.Addr {
	return c.Conn.LocalAddr()
}

// RemoteAddr implements the net.Conn RemoteAddr method.
func (c *WSConn) RemoteAddr() net.Addr {
	return c.Conn.RemoteAddr()
}

// SetDeadline implements the net.Conn SetDeadline method.
func (c *WSConn) SetDeadline(t time.Time) error {
	if err := c.Conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(t)
}

// SetReadDeadline implements the net.Conn SetReadDeadline method.
func (c *WSConn) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline implements the net.Conn SetWriteDeadline method.
func (c *WSConn) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}

// Dial connects to a provider endpoint, optionally using WebSocket fallback.
func Dial(ctx context.Context, endpoint string, tlsConfig *tls.Config, useWS bool) (net.Conn, error) {
	if !useWS {
		dialer := &tls.Dialer{Config: tlsConfig}
		return dialer.DialContext(ctx, "tcp", endpoint)
	}

	u := fmt.Sprintf("wss://%s/ws", endpoint)
	dialer := websocket.Dialer{
		TLSClientConfig:  tlsConfig,
		HandshakeTimeout: 10 * time.Second,
	}
	wsConn, _, err := dialer.Dial(u, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}
	return &WSConn{Conn: wsConn}, nil
}

// StartWSServer starts a WebSocket listener that upgrades connections and passes them to the handler.
func StartWSServer(ctx context.Context, addr string, tlsConfig *tls.Config, handler func(net.Conn, *http.Request)) error {
	upgrader := websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		handler(&WSConn{Conn: conn}, r)
	})

	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	err := server.ListenAndServeTLS("", "")
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
