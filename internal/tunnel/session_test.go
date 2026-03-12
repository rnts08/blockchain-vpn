package tunnel

import (
	"testing"
)

func TestSessionStats(t *testing.T) {
	stats := newSessionStats()
	if stats.startedAt.IsZero() {
		t.Error("startedAt is zero")
	}

	// Test upstream/downstream accounting
	stats.addUpstream(100)
	stats.addDownstream(200)
	if stats.upstreamBytes.Load() != 100 {
		t.Errorf("upstreamBytes = %d, want 100", stats.upstreamBytes.Load())
	}
	if stats.downstreamBytes.Load() != 200 {
		t.Errorf("downstreamBytes = %d, want 200", stats.downstreamBytes.Load())
	}

	stats.addUpstream(50)
	stats.addDownstream(50)
	if stats.upstreamBytes.Load() != 150 {
		t.Errorf("upstreamBytes after second add = %d, want 150", stats.upstreamBytes.Load())
	}
	if stats.downstreamBytes.Load() != 250 {
		t.Errorf("downstreamBytes after second add = %d, want 250", stats.downstreamBytes.Load())
	}
}

func TestRateEnforcer_NoLimit(t *testing.T) {
	re := newRateEnforcer(0) // 0 means no limit (should skip)
	re.accountAndThrottle(100)
	// Should not panic and transferred should still be 0 because early return
	if re.transferred != 0 {
		t.Errorf("transferred = %d, want 0", re.transferred)
	}
}

func TestRateEnforcer_Accounting(t *testing.T) {
	re := newRateEnforcer(1000) // 1000 bytes/sec
	initialTransferred := re.transferred

	// First transfer of 500 bytes
	re.accountAndThrottle(500)
	if re.transferred != initialTransferred+500 {
		t.Errorf("transferred = %d, want %d", re.transferred, initialTransferred+500)
	}

	// Second transfer of 300 bytes
	re.accountAndThrottle(300)
	if re.transferred != initialTransferred+800 {
		t.Errorf("transferred = %d, want %d", re.transferred, initialTransferred+800)
	}
}

func TestParseBandwidthLimit(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"0bit", 0, false},
		{"0bps", 0, false},
		{"100", 12, false}, // 100 bits/sec = 12 bytes/sec (integer division)
		{"100 bps", 12, false},
		{"100bit", 12, false},
		{"100bps", 12, false},
		{"1k", 125, false}, // 1000 bits/sec = 125 bytes/sec
		{"1kbit", 125, false},
		{"1kbps", 125, false},
		{"10K", 1250, false},  // 10,000 bits/sec = 1,250 bytes/sec
		{"1m", 125000, false}, // 1,000,000 bits/sec = 125,000 bytes/sec
		{"1Mbit", 125000, false},
		{"1Mbps", 125000, false},
		{"1g", 125000000, false}, // 1,000,000,000 bits/sec = 125,000,000 bytes/sec
		{"1Gbps", 125000000, false},
		{"  50  bps  ", 6, false}, // 50 bits/sec = 6 bytes/sec
		{"invalid", 0, true},
		{"123abc", 0, true},
		{"-100", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseBandwidthLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBandwidthLimit(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseBandwidthLimit(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestClientSessionConstruction verifies that clientSession struct is created correctly
func TestClientSessionConstruction(t *testing.T) {
	// Pass nil for conn - newClientSession doesn't immediately use it
	session := newClientSession(nil, 100000)

	if session == nil {
		t.Fatal("newClientSession returned nil")
	}
	if session.stats == nil {
		t.Error("stats is nil")
	}
	if session.upstreamLimiter == nil {
		t.Error("upstreamLimiter is nil")
	}
	if session.downLimiter == nil {
		t.Error("downLimiter is nil")
	}
	if session.upstreamLimiter.bytesPerSecond != 100000 {
		t.Errorf("upstreamLimiter.bytesPerSecond = %d, want 100000", session.upstreamLimiter.bytesPerSecond)
	}
	if session.downLimiter.bytesPerSecond != 100000 {
		t.Errorf("downLimiter.bytesPerSecond = %d, want 100000", session.downLimiter.bytesPerSecond)
	}
}
