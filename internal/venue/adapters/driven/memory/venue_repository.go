// Package memory holds in-memory driven adapters: real, working
// implementations of the Venue ports, used by fast tests and for running the
// app without Postgres.
package memory

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueRepository is an in-memory application.VenueRepository.
type VenueRepository struct{}

// NewVenueRepository returns an empty repository.
func NewVenueRepository() *VenueRepository { return &VenueRepository{} }

// Get implements application.VenueRepository.
func (r *VenueRepository) Get(_ context.Context, id domain.VenueID) (*domain.Venue, error) {
	return nil, fmt.Errorf("%w: %s", application.ErrVenueNotFound, id)
}
