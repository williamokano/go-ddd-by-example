package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// maxTitleLength is SHW-1's limit, in characters.
const maxTitleLength = 200

// Show is the aggregate root of Show Scheduling: one scheduled performance at
// a venue. It references the venue by ID only.
type Show struct {
	id         ShowID
	venueID    VenueID
	promoterID PromoterID
	title      string
	schedule   Schedule
	prices     PriceList
	status     Status
	version    int

	cancellationReason CancellationReason

	events Events
}

// DraftShow drafts a show at an active venue (SHW-1), starting in the future
// (SHW-2).
func DraftShow(id ShowID, venue VenueLayout, promoter PromoterID, title string, schedule Schedule, now time.Time) (*Show, error) {
	if id.IsZero() || promoter.IsZero() || venue.VenueID.IsZero() {
		return nil, fmt.Errorf("%w: zero show, venue or promoter id", ErrInvalidID)
	}
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > maxTitleLength {
		return nil, fmt.Errorf("%w: %q must be 1 to %d characters", ErrInvalidTitle, title, maxTitleLength)
	}
	if !venue.Active {
		return nil, fmt.Errorf("%w: venue %s", ErrVenueNotActive, venue.VenueID)
	}
	if err := startsInTheFuture(schedule, now); err != nil {
		return nil, err
	}
	s := &Show{id: id, venueID: venue.VenueID, promoterID: promoter, title: title, schedule: schedule, status: Draft}
	s.events.Record(ShowDrafted{ShowID: id, VenueID: venue.VenueID, PromoterID: promoter, Title: title, Schedule: schedule, At: now})
	return s, nil
}

func startsInTheFuture(s Schedule, now time.Time) error {
	if s.IsZero() {
		return fmt.Errorf("%w: no schedule", ErrInvalidSchedule)
	}
	if !s.StartsAt().After(now) {
		return fmt.Errorf("%w: starts at %s, not in the future", ErrInvalidSchedule, s.StartsAt().Format(time.RFC3339))
	}
	return nil
}

// Price sets the show's price list: one positive price for every section of
// the venue, and nothing else (SHW-4). Only a draft can be priced (SHW-5).
func (s *Show) Price(prices PriceList, venue VenueLayout, now time.Time) error {
	if s.status != Draft {
		return fmt.Errorf("%w: cannot price a %s show", ErrShowNotDraft, s.status)
	}
	if err := prices.CoversExactly(venue.SectionCodes()); err != nil {
		return err
	}
	s.prices = prices
	s.events.Record(ShowPriced{ShowID: s.id, Prices: prices, At: now})
	return nil
}

// Publish puts the show on sale. It needs a complete price list (SHW-5) and a
// start in the future (SHW-2). The recorded ShowPublished carries a snapshot
// of the layout and prices, so Ticketing never has to ask anyone.
func (s *Show) Publish(venue VenueLayout, now time.Time) error {
	if s.status != Draft {
		return fmt.Errorf("%w: cannot publish a %s show", ErrInvalidShowTransition, s.status)
	}
	if s.prices.IsZero() {
		return ErrShowNotPriced
	}
	if err := s.prices.CoversExactly(venue.SectionCodes()); err != nil {
		return err
	}
	if err := startsInTheFuture(s.schedule, now); err != nil {
		return err
	}
	s.status = Published
	s.events.Record(ShowPublished{ShowID: s.id, VenueID: s.venueID, Schedule: s.schedule, Layout: venue, Prices: s.prices, At: now})
	return nil
}

// Reschedule moves a draft show (SHW-5) to a new schedule in the future
// (SHW-2). Overlaps with other shows are the SchedulingPolicy's job (SHW-3).
func (s *Show) Reschedule(schedule Schedule, now time.Time) error {
	if s.status != Draft {
		return fmt.Errorf("%w: cannot reschedule a %s show", ErrShowNotDraft, s.status)
	}
	if err := startsInTheFuture(schedule, now); err != nil {
		return err
	}
	s.schedule = schedule
	s.events.Record(ShowRescheduled{ShowID: s.id, Schedule: schedule, At: now})
	return nil
}

// Cancel cancels a draft, published or sold-out show; Cancelled and Completed
// are terminal (SHW-6).
func (s *Show) Cancel(reason CancellationReason, now time.Time) error {
	if s.status.IsTerminal() {
		return fmt.Errorf("%w: cannot cancel a %s show", ErrInvalidShowTransition, s.status)
	}
	if reason == (CancellationReason{}) {
		return fmt.Errorf("%w: no reason", ErrInvalidCancellationReason)
	}
	s.status = Cancelled
	s.cancellationReason = reason
	s.events.Record(ShowCancelled{ShowID: s.id, VenueID: s.venueID, Reason: reason, At: now})
	return nil
}

// MarkSoldOut records Ticketing's fact that every seat is sold (SHW-8). The
// fact may arrive twice, or after the show was cancelled or completed: then
// it is old news, a no-op, not an error.
func (s *Show) MarkSoldOut(now time.Time) error {
	switch s.status {
	case Published:
		s.status = SoldOut
		s.events.Record(ShowSoldOut{ShowID: s.id, At: now})
		return nil
	case SoldOut, Cancelled, Completed:
		return nil
	default:
		return fmt.Errorf("%w: a %s show cannot sell out", ErrInvalidShowTransition, s.status)
	}
}

// ID returns the show's identity.
func (s *Show) ID() ShowID { return s.id }

// VenueID returns the venue the show takes place at.
func (s *Show) VenueID() VenueID { return s.venueID }

// PromoterID returns the promoter who drafted the show.
func (s *Show) PromoterID() PromoterID { return s.promoterID }

// Title returns the show's title.
func (s *Show) Title() string { return s.title }

// Schedule returns when the show takes place.
func (s *Show) Schedule() Schedule { return s.schedule }

// Prices returns the show's price list (zero until priced).
func (s *Show) Prices() PriceList { return s.prices }

// Status returns where the show is in its lifecycle.
func (s *Show) Status() Status { return s.status }

// CancellationReason says why the show was cancelled (zero unless cancelled).
func (s *Show) CancellationReason() CancellationReason { return s.cancellationReason }

// Version is the version the show was loaded at (ADR-011).
func (s *Show) Version() int { return s.version }

// PullEvents returns the recorded events and forgets them.
func (s *Show) PullEvents() []DomainEvent { return s.events.PullEvents() }
