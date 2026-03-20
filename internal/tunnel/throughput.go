package tunnel

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

// StartThroughputServer runs a simple TCP streaming server on the specified port.
// When a client connects, it immediately streams 2MB of zeroes and closes the connection.
func StartThroughputServer(ctx context.Context, port int) error {
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("throughput server failed to listen on %d: %w", port, err)
	}
	log.Printf("Throughput probe server listening on TCP %d", port)

	go func() {
		<-ctx.Done()
		l.Close()
	}()

	// 2MB fixed payload size for probes
	const payloadSize = 2 * 1024 * 1024
	zeroBuf := make([]byte, 32*1024)

	for {
		conn, err := l.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("Throughput server accept error: %v", err)
				continue
			}
		}

		go func(c net.Conn) {
			defer c.Close()
			_ = c.SetDeadline(time.Now().Add(5 * time.Second))

			bytesSent := 0
			for bytesSent < payloadSize {
				chunk := len(zeroBuf)
				if bytesSent+chunk > payloadSize {
					chunk = payloadSize - bytesSent
				}
				n, err := c.Write(zeroBuf[:chunk])
				if err != nil {
					return
				}
				bytesSent += n
			}
		}(conn)
	}
}

// MeasureProviderThroughputKbps connects to a provider's throughput port,
// downloads a fixed payload, and calculates the effective speed in Kbps.
func MeasureProviderThroughputKbps(ctx context.Context, endpoint string) (uint32, error) {
	start := time.Now()

	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to throughput endpoint: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, err
	}

	bytesRead, err := io.Copy(io.Discard, conn)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("read error during probe: %w", err)
	}

	duration := time.Since(start).Seconds()
	if duration <= 0 || bytesRead == 0 {
		return 0, fmt.Errorf("insufficient data transferred")
	}

	// Calculate bits per second, then convert to Kbps
	bits := float64(bytesRead) * 8
	kbps := uint32((bits / duration) / 1000)
	return kbps, nil
}

// MeasureLocalBandwidthKbps performs a self-contained TCP loopback bandwidth test
// to estimate the provider's maximum throughput capability. It starts a temporary
// server, connects to it, and measures how fast data can be transferred.
// Returns the measured bandwidth in Kbps.
func MeasureLocalBandwidthKbps(ctx context.Context) (uint32, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	const testPayloadSize = 2 * 1024 * 1024 // 2MB for accurate measurement
	const testTimeout = 5 * time.Second

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to create test listener: %w", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.SetDeadline(time.Now().Add(testTimeout))

		buf := make([]byte, 32*1024)
		sent := 0
		for sent < testPayloadSize {
			n, err := conn.Write(buf)
			if err != nil {
				return
			}
			sent += n
		}
	}()

	start := time.Now()
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to test server: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(testTimeout))

	received := 0
	for received < testPayloadSize {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		buf := make([]byte, 32*1024)
		n, err := conn.Read(buf)
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("read error during bandwidth test: %w", err)
		}
		if n == 0 {
			break
		}
		received += n
	}

	<-serverDone

	duration := time.Since(start).Seconds()
	if duration <= 0 || received == 0 {
		return 0, fmt.Errorf("insufficient data transferred in bandwidth test")
	}

	bits := float64(received) * 8
	kbps := uint32((bits / duration) / 1000)
	return kbps, nil
}
