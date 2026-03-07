package tunnel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeasureURLThroughputKbps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 512*1024)
		for i := range buf {
			buf[i] = 'a'
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf)
	}))
	defer srv.Close()

	kbps, err := measureURLThroughputKbps(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("measureURLThroughputKbps failed: %v", err)
	}
	if kbps == 0 {
		t.Fatalf("expected non-zero throughput")
	}
}
