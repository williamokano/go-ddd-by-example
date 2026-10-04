package application_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

func TestDraftShow(t *testing.T) {
	t.Run("drafts a show at a known, active venue", func(t *testing.T) {
		f := newFixture(t)
		venue := f.knownVenue(t)

		id, err := f.draft.Handle(f.ctx, f.draftCmd(venue, 30*24*time.Hour))

		if err != nil {
			t.Fatal(err)
		}
		if got := f.status(t, id); got != domain.Draft {
			t.Errorf("status = %v, want draft", got)
		}
	})

	t.Run("a venue Show never heard of is ErrVenueUnknown", func(t *testing.T) {
		f := newFixture(t)
		unknown, _ := domain.ParseVenueID(uuid.NewString())

		_, err := f.draft.Handle(f.ctx, f.draftCmd(unknown, 30*24*time.Hour))

		if !errors.Is(err, application.ErrVenueUnknown) {
			t.Errorf("error = %v, want %v", err, application.ErrVenueUnknown)
		}
	})

	t.Run("a retired venue is ErrVenueNotActive (SHW-1)", func(t *testing.T) {
		f := newFixture(t)
		venue := f.knownVenue(t)
		if err := application.NewOnVenueRetiredHandler(f.layouts, f.shows, f.clock).Handle(f.ctx, application.OnVenueRetired{VenueID: venue.String()}); err != nil {
			t.Fatal(err)
		}

		_, err := f.draft.Handle(f.ctx, f.draftCmd(venue, 30*24*time.Hour))

		if !errors.Is(err, domain.ErrVenueNotActive) {
			t.Errorf("error = %v, want %v", err, domain.ErrVenueNotActive)
		}
	})

	t.Run("an overlapping show is ErrScheduleConflict and nothing is saved (SHW-3)", func(t *testing.T) {
		f := newFixture(t)
		venue := f.knownVenue(t)
		if _, err := f.draft.Handle(f.ctx, f.draftCmd(venue, 30*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
		before := len(f.shows.Published())

		_, err := f.draft.Handle(f.ctx, f.draftCmd(venue, 30*24*time.Hour+time.Hour))

		if !errors.Is(err, domain.ErrScheduleConflict) || len(f.shows.Published()) != before {
			t.Errorf("error = %v, saved %d events; want ErrScheduleConflict, nothing saved", err, len(f.shows.Published())-before)
		}
	})

	t.Run("malformed input is the domain's validation error", func(t *testing.T) {
		f := newFixture(t)
		cmd := f.draftCmd(f.knownVenue(t), 30*24*time.Hour)
		cmd.PromoterID = "nope"

		if _, err := f.draft.Handle(f.ctx, cmd); !errors.Is(err, domain.ErrInvalidID) {
			t.Errorf("error = %v, want %v", err, domain.ErrInvalidID)
		}
	})
}

func TestPriceAndPublishShow(t *testing.T) {
	t.Run("publishing records ShowPublished with the layout snapshot", func(t *testing.T) {
		f := newFixture(t)

		id := f.publishedShow(t)

		if got := f.status(t, id); got != domain.Published {
			t.Errorf("status = %v, want published", got)
		}
		ev, ok := f.lastEvent(t).(domain.ShowPublished)
		if !ok || len(ev.Layout.Sections) != 2 || ev.Prices.IsZero() {
			t.Errorf("last event = %+v, want ShowPublished with layout and prices", f.lastEvent(t))
		}
	})

	t.Run("an incomplete price list is rejected (SHW-4)", func(t *testing.T) {
		f := newFixture(t)
		id := f.draftedShow(t)
		cmd := f.fullPrices(id)
		cmd.Prices = cmd.Prices[:1]

		if err := f.price.Handle(f.ctx, cmd); !errors.Is(err, domain.ErrPriceListMismatch) {
			t.Errorf("error = %v, want %v", err, domain.ErrPriceListMismatch)
		}
	})

	t.Run("publishing an unpriced show is ErrShowNotPriced (SHW-5)", func(t *testing.T) {
		f := newFixture(t)
		id := f.draftedShow(t)

		err := f.publish.Handle(f.ctx, application.PublishShow{ShowID: id.String(), PromoterID: f.promoter})

		if !errors.Is(err, domain.ErrShowNotPriced) {
			t.Errorf("error = %v, want %v", err, domain.ErrShowNotPriced)
		}
	})
}

// Only the promoter who drafted a show may change it. An application rule:
// it is about who may run the use case, not about what a show is.
func TestOnlyThePromoterMayActOnTheShow(t *testing.T) {
	f := newFixture(t)
	id := f.draftedShow(t)
	stranger := uuid.NewString()

	priceCmd := f.fullPrices(id)
	priceCmd.PromoterID = stranger
	errs := map[string]error{
		"price":   f.price.Handle(f.ctx, priceCmd),
		"publish": f.publish.Handle(f.ctx, application.PublishShow{ShowID: id.String(), PromoterID: stranger}),
		"cancel":  f.cancel.Handle(f.ctx, application.CancelShow{ShowID: id.String(), PromoterID: stranger, Reason: "mine now"}),
	}

	for name, err := range errs {
		if !errors.Is(err, application.ErrNotPromoter) {
			t.Errorf("%s by a stranger: error = %v, want %v", name, err, application.ErrNotPromoter)
		}
	}
}

func TestCancelShow(t *testing.T) {
	f := newFixture(t)
	id := f.publishedShow(t)

	err := f.cancel.Handle(f.ctx, application.CancelShow{ShowID: id.String(), PromoterID: f.promoter, Reason: "artist ill"})

	if err != nil {
		t.Fatal(err)
	}
	if ev, ok := f.lastEvent(t).(domain.ShowCancelled); !ok || ev.Reason.String() != "artist ill" {
		t.Errorf("last event = %+v", f.lastEvent(t))
	}
}

func TestMarkShowSoldOut(t *testing.T) {
	t.Run("a published show sells out; the same fact twice is a no-op (SHW-8)", func(t *testing.T) {
		f := newFixture(t)
		id := f.publishedShow(t)

		err1 := f.soldOut.Handle(f.ctx, application.MarkShowSoldOut{ShowID: id.String()})
		before := len(f.shows.Published())
		err2 := f.soldOut.Handle(f.ctx, application.MarkShowSoldOut{ShowID: id.String()})

		if err1 != nil || err2 != nil || f.status(t, id) != domain.SoldOut || len(f.shows.Published()) != before {
			t.Errorf("errs = %v, %v; status %v", err1, err2, f.status(t, id))
		}
	})

	t.Run("a cancelled show ignores the late fact (SHW-8)", func(t *testing.T) {
		f := newFixture(t)
		id := f.publishedShow(t)
		if err := f.cancel.Handle(f.ctx, application.CancelShow{ShowID: id.String(), PromoterID: f.promoter, Reason: "rain"}); err != nil {
			t.Fatal(err)
		}

		if err := f.soldOut.Handle(f.ctx, application.MarkShowSoldOut{ShowID: id.String()}); err != nil || f.status(t, id) != domain.Cancelled {
			t.Errorf("error = %v, status %v; want no-op", err, f.status(t, id))
		}
	})
}
