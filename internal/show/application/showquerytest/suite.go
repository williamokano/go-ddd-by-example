// Package showquerytest is the contract of application.ShowQueries.
package showquerytest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// Run runs the contract against queries reading what repo stored.
func Run(t *testing.T, newQueries func(t *testing.T) (application.ShowRepository, application.ShowQueries)) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

	t.Run("get renders a priced show", func(t *testing.T) {
		repo, queries := newQueries(t)
		layout := domain.VenueLayout{VenueID: domain.NewVenueID(uuid.New()), Active: true,
			Sections: []domain.LayoutSection{{Code: "FLOOR", Kind: "ga", Capacity: 3}}}
		start := now.Add(30 * 24 * time.Hour)
		schedule, _ := domain.NewSchedule(start.Add(-time.Hour), start, start.Add(2*time.Hour))
		promoter := domain.NewPromoterID(uuid.New())
		show, err := domain.DraftShow(domain.NewShowID(uuid.New()), layout, promoter, "Fado", schedule, now)
		if err != nil {
			t.Fatal(err)
		}
		eur, _ := sharedkernel.NewCurrency("EUR")
		price, _ := sharedkernel.NewMoney(2500, eur)
		prices, _ := domain.NewPriceList(map[string]sharedkernel.Money{"FLOOR": price})
		if err := show.Price(prices, layout, now); err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, show); err != nil {
			t.Fatal(err)
		}

		got, err := queries.Get(ctx, show.ID())

		if err != nil {
			t.Fatal(err)
		}
		want := application.ShowView{
			ID: show.ID().String(), VenueID: layout.VenueID.String(), PromoterID: promoter.String(), Title: "Fado",
			DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour), Status: "draft",
			Prices: []application.PriceView{{Section: "FLOOR", Amount: 2500, Currency: "EUR"}},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("view mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("an unknown show is ErrShowNotFound", func(t *testing.T) {
		_, queries := newQueries(t)

		if _, err := queries.Get(ctx, domain.NewShowID(uuid.New())); !errors.Is(err, application.ErrShowNotFound) {
			t.Errorf("error = %v, want %v", err, application.ErrShowNotFound)
		}
	})
}
