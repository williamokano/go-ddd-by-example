package postgres

import (
	"encoding/json"
	"fmt"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ToOutboxMessages translates Show's domain events into its Published
// Language. Only ShowPublished and ShowCancelled leave the context.
func ToOutboxMessages(events []sharedkernel.DomainEvent, newID func() uuid.UUID) ([]outbox.Message, error) {
	var msgs []outbox.Message
	for _, ev := range events {
		var (
			eventType string
			payload   any
			showID    domain.ShowID
		)
		switch e := ev.(type) {
		case domain.ShowPublished:
			eventType, showID = contracts.TypeShowPublishedV1, e.ShowID
			payload = contracts.ShowPublishedV1{
				ShowID: e.ShowID.String(), VenueID: e.VenueID.String(), Title: e.Title,
				DoorsOpen: e.Schedule.DoorsOpen(), StartsAt: e.Schedule.StartsAt(), EndsAt: e.Schedule.EndsAt(),
				Sections: pricedSections(e.Layout, e.Prices), PublishedAt: e.At,
			}
		case domain.ShowCancelled:
			eventType, showID = contracts.TypeShowCancelledV1, e.ShowID
			payload = contracts.ShowCancelledV1{
				ShowID: e.ShowID.String(), VenueID: e.VenueID.String(), Reason: e.Reason.String(), CancelledAt: e.At,
			}
		default:
			continue
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", eventType, err)
		}
		msgs = append(msgs, outbox.Message{
			EventID: newID(), Topic: contracts.Topic, Key: showID.String(),
			Type: eventType, Payload: b, OccurredAt: ev.OccurredAt(),
		})
	}
	return msgs, nil
}

func pricedSections(layout domain.VenueLayout, prices domain.PriceList) []contracts.SectionV1 {
	out := make([]contracts.SectionV1, 0, len(layout.Sections))
	for _, s := range layout.Sections {
		price, _ := prices.Price(s.Code)
		sv := contracts.SectionV1{
			Code: s.Code, Kind: s.Kind, Capacity: s.Capacity,
			Price: contracts.PriceV1{Amount: price.Amount(), Currency: price.Currency().String()},
		}
		for _, r := range s.Rows {
			sv.Rows = append(sv.Rows, contracts.RowV1{Label: r.Label, Seats: r.Seats})
		}
		out = append(out, sv)
	}
	return out
}
