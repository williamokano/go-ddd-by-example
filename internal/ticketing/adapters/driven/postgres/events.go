package postgres

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/outbox"
	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/sagamsg"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// ToOutboxMessages translates Ticketing's domain events into messages: the
// Published Language on ticketing.events (sold out, tickets issued, refunded)
// and the saga's private steps on ticketing.internal (keyed by order, so one
// order's steps stay in order). Everything else stays inside.
func ToOutboxMessages(events []sharedkernel.DomainEvent, newID func() uuid.UUID) ([]outbox.Message, error) {
	var msgs []outbox.Message
	add := func(topic, key, eventType string, payload any, ev sharedkernel.DomainEvent) error {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("%s: %w", eventType, err)
		}
		msgs = append(msgs, outbox.Message{EventID: newID(), Topic: topic, Key: key, Type: eventType, Payload: b, OccurredAt: ev.OccurredAt()})
		return nil
	}
	for _, ev := range events {
		var err error
		switch e := ev.(type) {
		case domain.OrderPaid:
			err = add(sagamsg.Topic, e.OrderID.String(), sagamsg.TypeOrderPaid,
				sagamsg.OrderPaid{OrderID: e.OrderID.String(), ShowID: e.ShowID.String(), Section: e.Section, HoldID: e.HoldID.String()}, ev)
		case domain.SeatsSold:
			seats := make([]string, len(e.Seats))
			for i, s := range e.Seats {
				seats[i] = s.String()
			}
			err = add(sagamsg.Topic, e.OrderID.String(), sagamsg.TypeSeatsSold,
				sagamsg.SeatsSold{OrderID: e.OrderID.String(), ShowID: e.ShowID.String(), Seats: seats}, ev)
		case domain.HoldConfirmationFailed:
			err = add(sagamsg.Topic, e.OrderID.String(), sagamsg.TypeHoldConfirmationFailed,
				sagamsg.HoldConfirmationFailed{OrderID: e.OrderID.String(), ShowID: e.ShowID.String(), Reason: e.Reason}, ev)
		case domain.InventoryClosed:
			err = add(sagamsg.Topic, e.ShowID.String(), sagamsg.TypeInventoryClosed, sagamsg.InventoryClosed{ShowID: e.ShowID.String()}, ev)
		case domain.SectionSoldOut:
			err = add(sagamsg.Topic, e.ShowID.String(), sagamsg.TypeSectionSoldOut,
				sagamsg.SectionSoldOut{ShowID: e.ShowID.String(), Section: e.Section}, ev)
		case domain.InventorySoldOut:
			err = add(contracts.Topic, e.ShowID.String(), contracts.TypeInventorySoldOutV1,
				contracts.InventorySoldOutV1{ShowID: e.ShowID.String(), SoldOutAt: e.At}, ev)
		case domain.OrderFulfilled:
			tickets := make([]contracts.TicketV1, len(e.Tickets))
			for i, tk := range e.Tickets {
				tickets[i] = contracts.TicketV1{Seat: tk.Seat.String(), Code: tk.Code.String()}
			}
			err = add(contracts.Topic, e.OrderID.String(), contracts.TypeTicketsIssuedV1, contracts.TicketsIssuedV1{
				OrderID: e.OrderID.String(), ShowID: e.ShowID.String(), CustomerID: e.CustomerID.String(),
				ContactEmail: e.ContactEmail.String(), Tickets: tickets, IssuedAt: e.At,
			}, ev)
		case domain.OrderRefunded:
			err = add(contracts.Topic, e.OrderID.String(), contracts.TypeOrderRefundedV1, contracts.OrderRefundedV1{
				OrderID: e.OrderID.String(), ShowID: e.ShowID.String(), CustomerID: e.CustomerID.String(),
				ContactEmail: e.ContactEmail.String(), Amount: e.Total.Amount(), Currency: e.Total.Currency().String(), RefundedAt: e.At,
			}, ev)
		}
		if err != nil {
			return nil, err
		}
	}
	return msgs, nil
}
