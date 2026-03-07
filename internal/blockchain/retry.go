package blockchain

import (
	"context"
	"fmt"
	"math/rand"
	"time"
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
		jitter := 0.8 + rand.Float64()*0.4
		sleep := time.Duration(float64(backoff) * jitter)
		select {
		case <-ctx.Done():
			return zero, fmt.Errorf("%s canceled: %w", op, ctx.Err())
		case <-time.After(sleep):
		}
		backoff *= 2
	}

	return zero, fmt.Errorf("%s failed after %d attempts: %w", op, attempts, lastErr)
}
