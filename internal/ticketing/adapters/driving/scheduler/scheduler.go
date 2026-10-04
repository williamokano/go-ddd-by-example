// Package scheduler is a driving adapter driven by time: on every tick it
// calls a use case, exactly like an HTTP handler or a Kafka consumer does on
// a request or a message.
package scheduler

import (
	"context"
	"log/slog"
	"time"
)

// Run calls job once per tick until ctx is done. Errors are logged; the next
// tick tries again. Production passes time.NewTicker(d).C; tests pass their
// own channel, so nothing sleeps.
func Run(ctx context.Context, ticks <-chan time.Time, job func(context.Context) error, logger *slog.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			if err := job(ctx); err != nil && ctx.Err() == nil {
				logger.WarnContext(ctx, "scheduled job failed", "error", err)
			}
		}
	}
}
