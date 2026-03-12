package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var (
	retryRandMu sync.Mutex
	retryRand   = rand.New(rand.NewSource(time.Now().UnixNano()))

	retryMetrics = &retryMetricsState{}
)

type retryMetricsState struct {
	totalRetries   atomic.Int64
	totalFailures  atomic.Int64
	lastRetryOp    atomic.Value // string
	retryLastError atomic.Value // string
	retryCountByOp sync.Map     // map[string]*atomic.Int64
}

func recordRetryMetrics(operation, errorMsg string) {
	retryMetrics.totalRetries.Add(1)

	// Track per-operation retries
	val, _ := retryMetrics.retryCountByOp.LoadOrStore(operation, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)

	retryMetrics.lastRetryOp.Store(operation)
	retryMetrics.retryLastError.Store(errorMsg)
}

func GetRetryMetricsSnapshot() map[string]any {
	retryOps := make(map[string]int64)
	retryMetrics.retryCountByOp.Range(func(key, value any) bool {
		retryOps[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	lastOp, _ := retryMetrics.lastRetryOp.Load().(string)
	lastErr, _ := retryMetrics.retryLastError.Load().(string)

	return map[string]any{
		"total_retries":        retryMetrics.totalRetries.Load(),
		"total_failures":       retryMetrics.totalFailures.Load(),
		"last_retry_op":        lastOp,
		"last_error":           lastErr,
		"retries_by_operation": retryOps,
	}
}

func withRetry[T any](ctx context.Context, op string, attempts int, initialBackoff time.Duration, fn func() (T, error)) (T, error) {
	var zero T
	if attempts < 1 {
		attempts = 1
	}
	if initialBackoff <= 0 {
		initialBackoff = 500 * time.Millisecond
	}

	backoff := initialBackoff
	var lastErr error
	for i := 0; i < attempts; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}
		lastErr = err

		// Record retry metrics
		recordRetryMetrics(op, err.Error())

		if i == attempts-1 {
			break
		}

		// +/-20% jitter to avoid sync retry storms.
		jitter := retryJitter()
		sleep := time.Duration(float64(backoff) * jitter)
		log.Printf("[retry] Retry %d/%d for %s after %vms (error: %v)", i+1, attempts, op, sleep.Milliseconds(), err)
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("%s canceled: %w", op, ctx.Err())
		case <-time.After(sleep):
		}
		backoff *= 2
	}

	return zero, fmt.Errorf("%s failed after %d attempts: %w", op, attempts, lastErr)
}

func retryJitter() float64 {
	retryRandMu.Lock()
	defer retryRandMu.Unlock()
	return 0.8 + retryRand.Float64()*0.4
}
