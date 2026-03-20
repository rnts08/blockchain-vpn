package tunnel

import (
	"context"
	"testing"
)

func TestMeasureLocalBandwidthKbps_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kbps, err := MeasureLocalBandwidthKbps(ctx)
	if err != nil {
		t.Fatalf("MeasureLocalBandwidthKbps failed: %v", err)
	}

	if kbps == 0 {
		t.Fatal("expected non-zero bandwidth measurement")
	}

	t.Logf("Measured bandwidth: %d Kbps", kbps)
}

func TestMeasureLocalBandwidthKbps_CancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := MeasureLocalBandwidthKbps(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestMeasureLocalBandwidthKbps_ReasonableRange(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kbps, err := MeasureLocalBandwidthKbps(ctx)
	if err != nil {
		t.Fatalf("MeasureLocalBandwidthKbps failed: %v", err)
	}

	if kbps < 1000 {
		t.Errorf("bandwidth unexpectedly low: %d Kbps (expected at least 1000 Kbps for loopback)", kbps)
	}
	if kbps > 10_000_000 {
		t.Errorf("bandwidth unexpectedly high: %d Kbps (unlikely even for loopback)", kbps)
	}
}
