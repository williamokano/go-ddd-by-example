package application

import (
	"context"
	"errors"
	"fmt"
)

// conflictAttempts bounds how often a use case re-runs after losing a race:
// on a hot show, holds serialise on one inventory version (ADR-005).
const conflictAttempts = 3

// RetryOnConflict re-runs the whole load → decide → save cycle while it fails
// with ErrConcurrentModification (ADR-011).
func RetryOnConflict(ctx context.Context, attempts int, fn func(context.Context) error) error {
	var err error
	for range attempts {
		err = fn(ctx)
		if !errors.Is(err, ErrConcurrentModification) {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("%w (after %w)", ctxErr, err)
		}
	}
	return err
}
