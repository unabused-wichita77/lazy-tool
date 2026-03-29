package connectors

import (
	"context"
	"time"

	"lazy-tool/internal/metrics"
)

const maxAttempts = 3

func withRetries(ctx context.Context, sourceID string, op func() error) error {
	var last error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			metrics.ConnectorRetry(sourceID, attempt, last)
			d := time.Duration(200*attempt) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(d):
			}
		}
		last = op()
		if last == nil {
			return nil
		}
	}
	return last
}
