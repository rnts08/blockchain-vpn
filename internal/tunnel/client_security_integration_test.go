//go:build functional

package tunnel

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestMeasureProviderThroughputKbps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Pick a random local port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close() // StartThroughputServer will bind it again

	go StartThroughputServer(ctx, port)

	// Give it a moment to bind
	time.Sleep(100 * time.Millisecond)

	endpoint := fmt.Sprintf("127.0.0.1:%d", port)
	kbps, err := MeasureProviderThroughputKbps(context.Background(), endpoint)
	if err != nil {
		t.Fatalf("MeasureProviderThroughputKbps failed: %v", err)
	}
	if kbps == 0 {
		t.Fatalf("expected non-zero throughput")
	}
}
