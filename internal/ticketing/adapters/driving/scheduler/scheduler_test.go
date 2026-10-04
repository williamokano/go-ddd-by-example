package scheduler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driving/scheduler"
)

func TestRun_CallsTheUseCaseOncePerTickAndStopsOnCancel(t *testing.T) {
	ticks := make(chan time.Time)
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		scheduler.Run(ctx, ticks, func(context.Context) error {
			calls.Add(1)
			return errors.New("logged, not fatal")
		}, slog.New(slog.NewTextHandler(io.Discard, nil)))
		close(done)
	}()

	ticks <- time.Now()
	ticks <- time.Now()
	ticks <- time.Now()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop on cancel")
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3 (one per tick)", calls.Load())
	}
}
