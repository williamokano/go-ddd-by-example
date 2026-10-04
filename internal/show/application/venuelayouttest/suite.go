// Package venuelayouttest is the contract of application.VenueLayouts.
package venuelayouttest

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// Run runs the contract.
func Run(t *testing.T, newLayouts func(t *testing.T) application.VenueLayouts) {
	t.Helper()
	ctx := context.Background()

	t.Run("an unknown venue is ErrVenueUnknown", func(t *testing.T) {
		_, err := newLayouts(t).Get(ctx, domain.NewVenueID(uuid.New()))

		if !errors.Is(err, application.ErrVenueUnknown) {
			t.Errorf("Get() error = %v, want %v", err, application.ErrVenueUnknown)
		}
	})

	t.Run("upsert then get round-trips, and a second upsert replaces", func(t *testing.T) {
		layouts := newLayouts(t)
		layout := domain.VenueLayout{
			VenueID: domain.NewVenueID(uuid.New()), Name: "Coliseu", Active: true, Country: "PT",
			Sections: []domain.LayoutSection{
				{Code: "ORCH", Kind: "seated", Rows: []domain.LayoutRow{{Label: "A", Seats: 10, Accessible: []int{1, 2}}}},
				{Code: "FLOOR", Kind: "ga", Capacity: 500},
			},
		}
		if err := layouts.Upsert(ctx, layout); err != nil {
			t.Fatal(err)
		}
		layout.Active = false
		if err := layouts.Upsert(ctx, layout); err != nil {
			t.Fatal(err)
		}

		got, err := layouts.Get(ctx, layout.VenueID)

		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(layout, got, cmp.AllowUnexported(domain.VenueID{})); diff != "" {
			t.Errorf("layout mismatch (-want +got):\n%s", diff)
		}
	})
}
