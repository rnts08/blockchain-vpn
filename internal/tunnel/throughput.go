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
