package domain

import "time"

// ShowSoldOut is a domain service: TKT-10 ("a show is sold out when every
// seat is sold") spans every SectionInventory of the show, so no single
// aggregate can decide it (ADR-013). It returns the event to announce, and
// whether the show is sold out. Deciding twice is harmless: Show treats a
// second InventorySoldOut as a no-op (SHW-8).
func ShowSoldOut(sections []*SectionInventory, now time.Time) (InventorySoldOut, bool) {
	if len(sections) == 0 {
		return InventorySoldOut{}, false
	}
	for _, s := range sections {
		if !s.IsSoldOut() {
			return InventorySoldOut{}, false
		}
	}
	return InventorySoldOut{ShowID: sections[0].ShowID(), At: now}, true
}
