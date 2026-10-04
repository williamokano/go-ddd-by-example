package application

import (
	"context"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowQueries is the read side: flat views, never aggregates.
type ShowQueries interface {
	// Get returns the view of one show, or ErrShowNotFound.
	Get(ctx context.Context, id domain.ShowID) (ShowView, error)
}

// ShowView is a show as readers see it.
type ShowView struct {
	ID                 string
	VenueID            string
	PromoterID         string
	Title              string
	DoorsOpen          time.Time
	StartsAt           time.Time
	EndsAt             time.Time
	Status             string
	CancellationReason string
	Prices             []PriceView
}

// PriceView is one section's price, in minor units.
type PriceView struct {
	Section  string
	Amount   int64
	Currency string
}

// NewShowView flattens a show. Query adapters that read the aggregate's
// stored state use it, so every implementation renders a show the same way.
func NewShowView(s *domain.Show) ShowView {
	v := ShowView{
		ID: s.ID().String(), VenueID: s.VenueID().String(), PromoterID: s.PromoterID().String(), Title: s.Title(),
		DoorsOpen: s.Schedule().DoorsOpen(), StartsAt: s.Schedule().StartsAt(), EndsAt: s.Schedule().EndsAt(),
		Status: s.Status().String(), CancellationReason: s.CancellationReason().String(),
	}
	for _, code := range s.Prices().Sections() {
		m, _ := s.Prices().Price(code)
		v.Prices = append(v.Prices, PriceView{Section: code, Amount: m.Amount(), Currency: m.Currency().String()})
	}
	return v
}
