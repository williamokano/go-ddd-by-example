package application

import (
	"context"

	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueQueries is the read side (CQRS-lite): flat views for screens and APIs,
// built straight from storage. It never returns aggregates: a reader can't
// call AddSection on a view, and listing venues doesn't load 1,000 aggregates.
type VenueQueries interface {
	// Get returns the view of one venue, or ErrVenueNotFound.
	Get(ctx context.Context, id domain.VenueID) (VenueView, error)

	// List returns the venues with the given status ("draft", "active",
	// "retired"), ordered by name.
	List(ctx context.Context, status string) ([]VenueView, error)
}

// VenueView is a venue as readers see it: strings and ints only.
type VenueView struct {
	ID       string
	Name     string
	Street   string
	City     string
	Country  string
	Status   string
	Capacity int
	Sections []SectionView
}

// SectionView is one section of a VenueView. Kind is "seated" or "ga".
type SectionView struct {
	Code     string
	Name     string
	Kind     string
	Capacity int
	Rows     []RowView
}

// RowView is one row of a seated SectionView.
type RowView struct {
	Label string
	Seats int
}
