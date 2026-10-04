package domain

// ShowState is everything a repository stores about a show.
type ShowState struct {
	ID                 ShowID
	VenueID            VenueID
	PromoterID         PromoterID
	Title              string
	Schedule           Schedule
	Prices             PriceList
	Status             Status
	CancellationReason CancellationReason
	Version            int
}

// RehydrateShow rebuilds a show from storage: no rules, no events. Only
// driven adapters may call it (the architecture test enforces that).
func RehydrateShow(s ShowState) *Show {
	return &Show{
		id: s.ID, venueID: s.VenueID, promoterID: s.PromoterID, title: s.Title,
		schedule: s.Schedule, prices: s.Prices, status: s.Status,
		cancellationReason: s.CancellationReason, version: s.Version,
	}
}
