package domain_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

var (
	now        = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	showID     = domain.NewShowID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000001"))
	venueID    = domain.NewVenueID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000002"))
	promoterID = domain.NewPromoterID(uuid.MustParse("0192f5e0-0000-7000-8000-000000000003"))

	showValues = cmp.AllowUnexported(domain.ShowID{}, domain.VenueID{}, domain.PromoterID{},
		domain.Schedule{}, domain.PriceList{}, domain.Money{}, domain.Currency{})
)

// activeLayout is Show's local view of an active venue with ORCH and FLOOR.
func activeLayout() domain.VenueLayout {
	return domain.VenueLayout{
		VenueID: venueID, Name: "Coliseu", Active: true,
		Sections: []domain.LayoutSection{
			{Code: "ORCH", Kind: "seated", Rows: []domain.LayoutRow{{Label: "A", Seats: 10}}},
			{Code: "FLOOR", Kind: "ga", Capacity: 500},
		},
	}
}

// inAMonth is a valid schedule starting 30 days after now.
func inAMonth(t *testing.T) domain.Schedule {
	t.Helper()
	start := now.Add(30 * 24 * time.Hour)
	s, err := domain.NewSchedule(start.Add(-time.Hour), start, start.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func fullPrices(t *testing.T) domain.PriceList {
	t.Helper()
	return priceList(t, map[string]domain.Money{"ORCH": eur(t, 4500), "FLOOR": eur(t, 2500)})
}

func pricedShow(t *testing.T) *domain.Show {
	t.Helper()
	s := draftShow(t)
	if err := s.Price(fullPrices(t), activeLayout(), now); err != nil {
		t.Fatal(err)
	}
	return s
}

func draftShow(t *testing.T) *domain.Show {
	t.Helper()
	s, err := domain.DraftShow(showID, activeLayout(), promoterID, "Fado Night", inAMonth(t), now)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
