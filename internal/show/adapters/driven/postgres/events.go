package postgres

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/show/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ToOutboxMessages translates Show's domain events into its Published
// Language. Only ShowPublished (v2; v1 is retired, 9.3) and ShowCancelled
// leave the context.
func ToOutboxMessages(events []sharedkernel.DomainEvent, newID func() uuid.UUID) ([]outbox.Message, error) {
	var msgs []outbox.Message
	add := func(showID domain.ShowID, eventType string, payload any, ev sharedkernel.DomainEvent) error {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%s: %w", eventType, err)
		}
		msgs = append(msgs, outbox.Message{
			EventID: newID(), Topic: contracts.Topic, Key: showID.String(),
			Type: eventType, Payload: b, OccurredAt: ev.OccurredAt(),
		})
		return nil
	}
	for _, ev := range events {
		var err error
		switch e := ev.(type) {
		case domain.ShowPublished:
			err = add(e.ShowID, contracts.TypeShowPublishedV2, contracts.ShowPublishedV2{
				ShowID: e.ShowID.String(), VenueID: e.VenueID.String(), VenueCountry: e.Layout.Country, Title: e.Title,
				DoorsOpen: e.Schedule.DoorsOpen(), StartsAt: e.Schedule.StartsAt(), EndsAt: e.Schedule.EndsAt(),
				Sections: pricedSectionsV2(e.Layout, e.Prices), PublishedAt: e.At,
			}, ev)
		case domain.ShowCancelled:
			err = add(e.ShowID, contracts.TypeShowCancelledV1, contracts.ShowCancelledV1{
				ShowID: e.ShowID.String(), VenueID: e.VenueID.String(), Reason: e.Reason.String(), CancelledAt: e.At,
			}, ev)
		}
		if err != nil {
			return nil, err
		}
	}
	return msgs, nil
}

func pricedSectionsV2(layout domain.VenueLayout, prices domain.PriceList) []contracts.SectionV2 {
	out := make([]contracts.SectionV2, 0, len(layout.Sections))
	for _, s := range layout.Sections {
		price, _ := prices.Price(s.Code)
		sv := contracts.SectionV2{
			Code: s.Code, Kind: s.Kind, Capacity: s.Capacity,
			Price: contracts.PriceV1{Amount: price.Amount(), Currency: price.Currency().String()},
		}
		for _, r := range s.Rows {
			for n := 1; n <= r.Seats; n++ {
				sv.Seats = append(sv.Seats, contracts.SeatV2{Row: r.Label, Number: n, Accessible: slices.Contains(r.Accessible, n)})
			}
		}
		out = append(out, sv)
	}
	return out
}
