package blockchain

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

var (
	retryRandMu sync.Mutex
	retryRand   = rand.New(rand.NewSource(time.Now().UnixNano()))
)

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
