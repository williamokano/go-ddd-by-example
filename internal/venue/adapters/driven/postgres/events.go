package postgres

import (
	"encoding/json"
	"fmt"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/venue/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// ToOutboxMessages translates domain events into integration events: a small
// anti-corruption layer pointing outward. It decides what leaves the context:
// VenueRegistered and SectionAdded are internal facts and are not published.
func ToOutboxMessages(events []sharedkernel.DomainEvent, newID func() uuid.UUID) ([]outbox.Message, error) {
	var msgs []outbox.Message
	for _, ev := range events {
		var (
			eventType string
			payload   any
			venueID   domain.VenueID
		)
		switch e := ev.(type) {
		case domain.VenueActivated:
			eventType, venueID = contracts.TypeVenueActivatedV1, e.VenueID
			payload = contracts.VenueActivatedV1{
				VenueID: e.VenueID.String(), Name: e.Name, Sections: sectionsV1(e.Sections), ActivatedAt: e.At,
			}
		case domain.VenueRetired:
			eventType, venueID = contracts.TypeVenueRetiredV1, e.VenueID
			payload = contracts.VenueRetiredV1{VenueID: e.VenueID.String(), RetiredAt: e.At}
		default:
			continue
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", eventType, err)
		}
		msgs = append(msgs, outbox.Message{
			EventID: newID(), Topic: contracts.Topic, Key: venueID.String(),
			Type: eventType, Payload: b, OccurredAt: ev.OccurredAt(),
		})
	}
	return msgs, nil
}

func sectionsV1(sections []domain.Section) []contracts.SectionV1 {
	out := make([]contracts.SectionV1, 0, len(sections))
	for _, s := range sections {
		sv := contracts.SectionV1{Code: s.Code().String(), Kind: contracts.KindSeated}
		if s.Kind() == domain.GeneralAdmission {
			sv.Kind, sv.Capacity = contracts.KindGA, s.Capacity()
		}
		for _, r := range s.Rows() {
			sv.Rows = append(sv.Rows, contracts.RowV1{Label: r.Label(), Seats: r.Seats()})
		}
		out = append(out, sv)
	}
	return out
}
