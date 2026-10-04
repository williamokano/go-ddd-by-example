package application

import (
	"context"
	"errors"
	"fmt"
)

// RetryOnConflict runs fn up to attempts times while it fails with
// ErrConcurrentModification.
//
// fn must be the whole load → decide → save cycle. Retrying only the save
// would write a decision taken on stale state; reloading lets the domain
// decide again against what is really stored (ADR-011).
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
