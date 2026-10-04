package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestRetryOnConflict(t *testing.T) {
	t.Run("retries the whole function after a conflict", func(t *testing.T) {
		calls := 0

		err := application.RetryOnConflict(context.Background(), 3, func(context.Context) error {
			calls++
			if calls == 1 {
				return application.ErrConcurrentModification
			}
			return nil
		})

		if err != nil || calls != 2 {
			t.Errorf("err = %v, calls = %d; want nil after 2 calls", err, calls)
		}
	})

	t.Run("gives up after the given attempts", func(t *testing.T) {
		calls := 0

		err := application.RetryOnConflict(context.Background(), 3, func(context.Context) error {
			calls++
			return application.ErrConcurrentModification
		})

		if !errors.Is(err, application.ErrConcurrentModification) || calls != 3 {
			t.Errorf("err = %v, calls = %d; want ErrConcurrentModification after 3 calls", err, calls)
		}
	})

	t.Run("does not retry other errors", func(t *testing.T) {
		calls := 0

		err := application.RetryOnConflict(context.Background(), 3, func(context.Context) error {
			calls++
			return domain.ErrVenueNotDraft
		})

		if !errors.Is(err, domain.ErrVenueNotDraft) || calls != 1 {
			t.Errorf("err = %v, calls = %d; want ErrVenueNotDraft after 1 call", err, calls)
		}
	})

	t.Run("stops when the context is done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0

		err := application.RetryOnConflict(ctx, 3, func(context.Context) error {
			calls++
			cancel()
			return application.ErrConcurrentModification
		})

		if !errors.Is(err, context.Canceled) || calls != 1 {
			t.Errorf("err = %v, calls = %d; want context.Canceled after 1 call", err, calls)
		}
	})
}

// conflictOnce is a VenueRepository whose first Save loses a race.
type conflictOnce struct {
	application.VenueRepository
	conflicted bool
}

func (r *conflictOnce) Save(ctx context.Context, v *domain.Venue) error {
	if !r.conflicted {
		r.conflicted = true
		return application.ErrConcurrentModification
	}
	return r.VenueRepository.Save(ctx, v)
}

func TestAddSection_RetriesAConflict(t *testing.T) {
	f := newFixture(t)
	id := f.draftVenue(t)
	handler := application.NewAddSectionHandler(&conflictOnce{VenueRepository: f.repo}, f.clock)

	err := handler.Handle(f.ctx, application.AddSection{VenueID: id.String(), Code: "BOX", Kind: application.KindGA, Capacity: 4})

	if err != nil {
		t.Fatalf("Handle() error = %v, want the retry to succeed", err)
	}
	if got := len(f.venue(t, id).Sections()); got != 2 {
		t.Errorf("venue has %d sections, want 2", got)
	}
}
