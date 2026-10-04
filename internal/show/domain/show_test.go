package domain_test

import (
	"errors"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func TestDraftShow(t *testing.T) {
	t.Run("a drafted show is a draft and records ShowDrafted", func(t *testing.T) {
		schedule := inAMonth(t)

		s, err := domain.DraftShow(showID, activeLayout(), promoterID, "  Fado Night ", schedule, now)

		if err != nil {
			t.Fatalf("DraftShow() error = %v", err)
		}
		if s.Status() != domain.Draft || s.Title() != "Fado Night" || s.VenueID() != venueID || s.PromoterID() != promoterID {
			t.Errorf("show = %v %q %v %v", s.Status(), s.Title(), s.VenueID(), s.PromoterID())
		}
		want := []sharedkernel.DomainEvent{domain.ShowDrafted{
			ShowID: showID, VenueID: venueID, PromoterID: promoterID, Title: "Fado Night", Schedule: schedule, At: now,
		}}
		if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	pastStart := func(t *testing.T) domain.Schedule {
		start := now.Add(-time.Hour)
		s, err := domain.NewSchedule(start, start, start.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	inactive := activeLayout()
	inactive.Active = false

	tests := []struct {
		name     string
		layout   domain.VenueLayout
		title    string
		schedule func(*testing.T) domain.Schedule
		wantErr  error
	}{
		{"blank title (SHW-1)", activeLayout(), " ", inAMonth, domain.ErrInvalidTitle},
		{"title over 200 characters (SHW-1)", activeLayout(), strings.Repeat("x", 201), inAMonth, domain.ErrInvalidTitle},
		{"inactive venue (SHW-1)", inactive, "Fado", inAMonth, domain.ErrVenueNotActive},
		{"start in the past (SHW-2)", activeLayout(), "Fado", pastStart, domain.ErrInvalidSchedule},
	}
	for _, tt := range tests {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			_, err := domain.DraftShow(showID, tt.layout, promoterID, tt.title, tt.schedule(t), now)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestShow_Price(t *testing.T) {
	t.Run("records ShowPriced (SHW-4, SHW-5)", func(t *testing.T) {
		s := draftShow(t)
		s.PullEvents()

		err := s.Price(fullPrices(t), activeLayout(), now)

		if err != nil {
			t.Fatalf("Price() error = %v", err)
		}
		want := []sharedkernel.DomainEvent{domain.ShowPriced{ShowID: showID, Prices: fullPrices(t), At: now}}
		if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("an incomplete price list is rejected (SHW-4)", func(t *testing.T) {
		s := draftShow(t)

		err := s.Price(priceList(t, map[string]sharedkernel.Money{"ORCH": eur(t, 4500)}), activeLayout(), now)

		if !errors.Is(err, domain.ErrPriceListMismatch) {
			t.Errorf("error = %v, want %v", err, domain.ErrPriceListMismatch)
		}
	})
}

func TestShow_Publish(t *testing.T) {
	t.Run("without a price list fails (SHW-5)", func(t *testing.T) {
		s := draftShow(t)

		err := s.Publish(activeLayout(), now)

		if !errors.Is(err, domain.ErrShowNotPriced) {
			t.Errorf("error = %v, want %v", err, domain.ErrShowNotPriced)
		}
	})

	t.Run("a priced show is published with the layout and prices snapshot", func(t *testing.T) {
		s := pricedShow(t)
		s.PullEvents()

		err := s.Publish(activeLayout(), now)

		if err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
		if s.Status() != domain.Published {
			t.Errorf("Status() = %v, want published", s.Status())
		}
		want := []sharedkernel.DomainEvent{domain.ShowPublished{
			ShowID: showID, VenueID: venueID, Title: "Fado Night", Schedule: inAMonth(t), Layout: activeLayout(), Prices: fullPrices(t), At: now,
		}}
		if diff := cmp.Diff(want, s.PullEvents(), showValues); diff != "" {
			t.Errorf("events mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("once the start has passed fails (SHW-2)", func(t *testing.T) {
		s := pricedShow(t)

		err := s.Publish(activeLayout(), s.Schedule().StartsAt())

		if !errors.Is(err, domain.ErrInvalidSchedule) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidSchedule)
		}
	})
}
