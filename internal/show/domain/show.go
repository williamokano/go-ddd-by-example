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
	status     Status
	version    int

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

// Status returns where the show is in its lifecycle.
func (s *Show) Status() Status { return s.status }

// Version is the version the show was loaded at (ADR-011).
func (s *Show) Version() int { return s.version }

// PullEvents returns the recorded events and forgets them.
func (s *Show) PullEvents() []DomainEvent { return s.events.PullEvents() }
