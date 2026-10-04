package domain

// VenueLayout is Show's own read-only copy of what it needs to know about a
// venue: whether it is active, and its layout (forwarded to Ticketing on
// publish). It is built from Venue's integration events by the anti-corruption
// layer, never by asking Venue.
type VenueLayout struct {
	VenueID  VenueID
	Name     string
	Active   bool
	Sections []LayoutSection
}

// LayoutSection is one section of the venue: rows when seated, a capacity
// when general admission ("ga").
type LayoutSection struct {
	Code     string
	Kind     string
	Rows     []LayoutRow
	Capacity int
}

// LayoutRow is one row of a seated section.
type LayoutRow struct {
	Label      string
	Seats      int
	Accessible []int // seat numbers with step-free access, sorted
}

// SectionCodes lists the venue's section codes, in layout order.
func (l VenueLayout) SectionCodes() []string {
	codes := make([]string, len(l.Sections))
	for i, s := range l.Sections {
		codes[i] = s.Code
	}
	return codes
}
