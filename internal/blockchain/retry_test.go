package blockchain

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWithRetry_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	result, err := withRetry(context.Background(), "TestOp", 3, 10*time.Millisecond, func() (int, error) {
		callCount++
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestWithRetry_FailureAfterRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	expectedErr := errors.New("test error")
	_, err := withRetry(context.Background(), "TestOp", 3, 10*time.Millisecond, func() (int, error) {
		callCount++
		return 0, expectedErr
	})

	if err == nil {
		t.Error("expected error")
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestWithRetry_CanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	callCount := 0
	_, err := withRetry(ctx, "TestOp", 3, 10*time.Millisecond, func() (int, error) {
		callCount++
		return 0, errors.New("error")
	})

	if err == nil {
		t.Error("expected error")
	}
	// Note: the first call always executes before context is checked between retries
	// The context is only checked after the first failure
	if callCount != 1 {
		t.Errorf("expected 1 call (first call always runs), got %d", callCount)
	}
}

func TestWithRetry_InvalidAttempts(t *testing.T) {
	t.Parallel()

	// Test with 0 attempts - should still work (minimum 1)
	callCount := 0
	result, err := withRetry(context.Background(), "TestOp", 0, 10*time.Millisecond, func() (int, error) {
		callCount++
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestWithRetry_InvalidBackoff(t *testing.T) {
	t.Parallel()

	// Test with 0 backoff - should use default
	callCount := 0
	result, err := withRetry(context.Background(), "TestOp", 2, 0, func() (int, error) {
		callCount++
		if callCount == 1 {
			return 0, errors.New("error")
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestRetryJitter(t *testing.T) {
	t.Parallel()

	// Test that jitter produces values between 0.8 and 1.2
	for i := 0; i < 100; i++ {
		j := retryJitter()
		if j < 0.8 || j > 1.2 {
			t.Errorf("jitter out of range: %f", j)
		}
	}
}

func TestRetryMetricsRecording(t *testing.T) {
	t.Parallel()

	// Get initial metrics
	initial := GetRetryMetricsSnapshot()
	initialRetries := initial["total_retries"].(int64)

	// Trigger a retry
	_, _ = withRetry(context.Background(), "MetricsTestOp", 2, 10*time.Millisecond, func() (int, error) {
		return 0, errors.New("test error for metrics")
	})

	// Check metrics were recorded
	after := GetRetryMetricsSnapshot()
	afterRetries := after["total_retries"].(int64)

	if afterRetries <= initialRetries {
		t.Errorf("expected retries to increase, was %d now %d", initialRetries, afterRetries)
	}

	// Check operation is tracked
	byOp := after["retries_by_operation"].(map[string]int64)
	if count, ok := byOp["MetricsTestOp"]; !ok || count == 0 {
		t.Errorf("expected MetricsTestOp in retries_by_operation, got %v", byOp)
	}
}

func TestGetRetryMetricsSnapshot(t *testing.T) {
	t.Parallel()

	// Test that snapshot returns expected structure
	snapshot := GetRetryMetricsSnapshot()

	if _, ok := snapshot["total_retries"]; !ok {
		t.Error("expected total_retries in snapshot")
	}
	if _, ok := snapshot["total_failures"]; !ok {
		t.Error("expected total_failures in snapshot")
	}
	if _, ok := snapshot["retries_by_operation"]; !ok {
		t.Error("expected retries_by_operation in snapshot")
	}
}
