package tunnel

import (
	"bytes"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestCopyStreamWithControl_ForwardsAndAccounts(t *testing.T) {
	reader, writer := net.Pipe()
	defer reader.Close()
	defer writer.Close()

	payload := bytes.Repeat([]byte{0x42}, 8192)
	var out bytes.Buffer
	var counted atomic.Int64

	done := make(chan struct{})
	go func() {
		copyStreamWithControl(&out, reader, func(n int) {
			counted.Add(int64(n))
		}, nil)
		close(done)
	}()

	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	_ = writer.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("copyStreamWithControl did not finish")
	}

	if !bytes.Equal(out.Bytes(), payload) {
		t.Fatalf("forwarded payload mismatch: got %d bytes want %d", out.Len(), len(payload))
	}
	if counted.Load() != int64(len(payload)) {
		t.Fatalf("accounted bytes mismatch: got %d want %d", counted.Load(), len(payload))
	}
}

func TestCopyStreamWithControl_EnforcesRate(t *testing.T) {
	reader, writer := net.Pipe()
	defer reader.Close()
	defer writer.Close()

	payload := bytes.Repeat([]byte{0x11}, 1024)
	limiter := newRateEnforcer(512) // bytes/sec

	done := make(chan struct{})
	start := time.Now()
	go func() {
		copyStreamWithControl(&bytes.Buffer{}, reader, nil, limiter)
		close(done)
	}()

	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	_ = writer.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("copyStreamWithControl did not finish with limiter")
	}

	if elapsed := time.Since(start); elapsed < 1800*time.Millisecond {
		t.Fatalf("rate limit not enforced; elapsed=%v expected >= ~2s", elapsed)
	}
}
