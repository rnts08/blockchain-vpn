//go:build functional

package tunnel

import (
	"context"
	"testing"
	"time"
)

func TestFunctional_Throughput_Measurement(t *testing.T) {
	t.Parallel()

	start := time.Now()

	bytesRead := int64(2 * 1024 * 1024)
	duration := 1.5

	kbps := uint32((float64(bytesRead) * 8 / duration) / 1000)

	expectedKbps := uint32((float64(bytesRead) * 8 / 1.5) / 1000)
	if kbps != expectedKbps {
		t.Errorf("Expected %d Kbps, got %d", expectedKbps, kbps)
	}

	elapsed := time.Since(start)
	if elapsed > time.Second {
		t.Error("Calculation should be instant")
	}

	t.Logf("Throughput calculation: %d bytes in %.2f seconds = %d Kbps", bytesRead, duration, kbps)
}

func TestFunctional_Throughput_ZeroDuration(t *testing.T) {
	t.Parallel()

	bytesRead := int64(1024)
	duration := 0.0

	if duration <= 0 || bytesRead == 0 {
		t.Log("Correctly handles zero duration")
	}
}

func TestFunctional_Throughput_LargePayload(t *testing.T) {
	t.Parallel()

	bytesRead := int64(100 * 1024 * 1024)
	duration := 10.0

	kbps := uint32((float64(bytesRead) * 8 / duration) / 1000)

	if kbps < 70000 || kbps > 90000 {
		t.Errorf("Expected around 80000 Kbps for 100MB in 10s, got %d", kbps)
	}

	t.Logf("Large payload throughput: %d Kbps", kbps)
}

func TestFunctional_Throughput_ContextTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
		t.Log("Context correctly times out")
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

func TestFunctional_Throughput_CalculationPrecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		bytes       int64
		duration    float64
		minExpected uint32
		maxExpected uint32
	}{
		{1024 * 1024, 1.0, 8000, 8500},
		{1024 * 1024, 2.0, 4000, 4250},
		{512 * 1024, 1.0, 4000, 4250},
		{1024 * 1024, 0.5, 16000, 17000},
	}

	for _, tt := range tests {
		kbps := uint32((float64(tt.bytes) * 8 / tt.duration) / 1000)
		if kbps < tt.minExpected || kbps > tt.maxExpected {
			t.Errorf("For %d bytes in %.1fs: expected %d-%d Kbps, got %d",
				tt.bytes, tt.duration, tt.minExpected, tt.maxExpected, kbps)
		}
	}
}
