package domain_test

import (
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
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
		domain.Schedule{}, domain.PriceList{}, sharedkernel.Money{}, sharedkernel.Currency{}, domain.CancellationReason{})
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
	return priceList(t, map[string]sharedkernel.Money{"ORCH": eur(t, 4500), "FLOOR": eur(t, 2500)})
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

// showIn rehydrates a priced show in the given status.
func showIn(t *testing.T, status domain.Status) *domain.Show {
	t.Helper()
	return domain.RehydrateShow(domain.ShowState{
		ID: showID, VenueID: venueID, PromoterID: promoterID, Title: "Fado Night",
		Schedule: inAMonth(t), Prices: fullPrices(t), Status: status, Version: 3,
	})
}

func reason(t *testing.T, r string) domain.CancellationReason {
	t.Helper()
	cr, err := domain.NewCancellationReason(r)
	if err != nil {
		t.Fatal(err)
	}
	return cr
}

func eur(t *testing.T, minor int64) sharedkernel.Money {
	t.Helper()
	return mustMoney(t, minor, "EUR")
}

func mustCurrency(t *testing.T, code string) sharedkernel.Currency {
	t.Helper()
	c, err := sharedkernel.NewCurrency(code)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
