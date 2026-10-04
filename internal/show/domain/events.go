package domain

import "time"

// ShowDrafted records that a promoter drafted a show at a venue.
type ShowDrafted struct {
	ShowID     ShowID
	VenueID    VenueID
	PromoterID PromoterID
	Title      string
	Schedule   Schedule
	At         time.Time
}

// EventName implements DomainEvent.
func (ShowDrafted) EventName() string { return "show.ShowDrafted" }

// OccurredAt implements DomainEvent.
func (e ShowDrafted) OccurredAt() time.Time { return e.At }

// ShowPriced records the show's price list.
type ShowPriced struct {
	ShowID ShowID
	Prices PriceList
	At     time.Time
}

// EventName implements DomainEvent.
func (ShowPriced) EventName() string { return "show.ShowPriced" }

// OccurredAt implements DomainEvent.
func (e ShowPriced) OccurredAt() time.Time { return e.At }

// ShowPublished records that the show went on sale, with the layout and
// prices it was published with.
type ShowPublished struct {
	ShowID   ShowID
	VenueID  VenueID
	Title    string
	Schedule Schedule
	Layout   VenueLayout
	Prices   PriceList
	At       time.Time
}

// EventName implements DomainEvent.
func (ShowPublished) EventName() string { return "show.ShowPublished" }

// OccurredAt implements DomainEvent.
func (e ShowPublished) OccurredAt() time.Time { return e.At }

// ShowRescheduled records that a draft show moved to a new schedule.
type ShowRescheduled struct {
	ShowID   ShowID
	Schedule Schedule
	At       time.Time
}

// EventName implements DomainEvent.
func (ShowRescheduled) EventName() string { return "show.ShowRescheduled" }

// OccurredAt implements DomainEvent.
func (e ShowRescheduled) OccurredAt() time.Time { return e.At }

// ShowCancelled records that the show was cancelled, and why.
type ShowCancelled struct {
	ShowID  ShowID
	VenueID VenueID
	Reason  CancellationReason
	At      time.Time
}

// EventName implements DomainEvent.
func (ShowCancelled) EventName() string { return "show.ShowCancelled" }

// OccurredAt implements DomainEvent.
func (e ShowCancelled) OccurredAt() time.Time { return e.At }

// ShowSoldOut records that every seat of the show is sold.
type ShowSoldOut struct {
	ShowID ShowID
	At     time.Time
}

// EventName implements DomainEvent.
func (ShowSoldOut) EventName() string { return "show.ShowSoldOut" }

// OccurredAt implements DomainEvent.
func (e ShowSoldOut) OccurredAt() time.Time { return e.At }

// ShowCompleted records that a show has ended (SHW-9).
type ShowCompleted struct {
	ShowID ShowID
	At     time.Time
}

// EventName implements DomainEvent.
func (ShowCompleted) EventName() string { return "show.ShowCompleted" }

// OccurredAt implements DomainEvent.
func (e ShowCompleted) OccurredAt() time.Time { return e.At }

// ShowBackOnSale records that a sold-out show has seats again (SHW-10).
type ShowBackOnSale struct {
	ShowID ShowID
	At     time.Time
}

// EventName implements DomainEvent.
func (ShowBackOnSale) EventName() string { return "show.ShowBackOnSale" }

// OccurredAt implements DomainEvent.
func (e ShowBackOnSale) OccurredAt() time.Time { return e.At }
