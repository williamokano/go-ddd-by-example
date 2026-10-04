package application

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueRepository loads and saves whole Venue aggregates (one per root, not
// one per table). It is a driven port: the core declares it, an adapter
// implements it (memory for tests, Postgres in production).
type VenueRepository interface {
	// Get loads the venue with the given ID, or returns ErrVenueNotFound.
	Get(ctx context.Context, id domain.VenueID) (*domain.Venue, error)

	// Save persists the aggregate and its pending events atomically: it drains
	// v.PullEvents() and stores them with the venue (the outbox, from Part 5).
	// It returns ErrConcurrentModification if the stored version differs from
	// v.Version(), i.e. someone else saved the venue since it was loaded.
	Save(ctx context.Context, v *domain.Venue) error
}

// Clock tells the time. The domain never calls time.Now(); use cases ask the
// clock and pass the time in (ADR-008).
type Clock interface {
	Now() time.Time
}

// IDGenerator hands out new identities. It is per context and per aggregate,
// so the type system stops a ShowID being used as a VenueID (ADR-008).
type IDGenerator interface {
	NewVenueID() domain.VenueID
}
