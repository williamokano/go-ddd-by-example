package application

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowRepository loads and saves Show aggregates.
type ShowRepository interface {
	// Get loads a show, or returns ErrShowNotFound.
	Get(ctx context.Context, id domain.ShowID) (*domain.Show, error)

	// Save persists the show and its integration events atomically; a stale
	// version is ErrConcurrentModification (ADR-011).
	Save(ctx context.Context, s *domain.Show) error

	// ListOpenAtVenue returns the venue's shows that are not cancelled or
	// completed: the ones the SchedulingPolicy and SHW-7 care about.
	ListOpenAtVenue(ctx context.Context, venueID domain.VenueID) ([]*domain.Show, error)
}

// VenueLayouts is Show's local projection of Venue's facts (the ACL's store).
type VenueLayouts interface {
	// Get returns a venue's layout, or ErrVenueUnknown if Show never heard of it.
	Get(ctx context.Context, id domain.VenueID) (domain.VenueLayout, error)

	// Upsert stores the layout, replacing any previous one (idempotent).
	Upsert(ctx context.Context, layout domain.VenueLayout) error
}

// Clock tells the time (ADR-008).
type Clock interface {
	Now() time.Time
}

// IDGenerator hands out new show identities.
type IDGenerator interface {
	NewShowID() domain.ShowID
}
