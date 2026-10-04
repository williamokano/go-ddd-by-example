// Package consumer drives Notifications from ticketing.events (group
// "notifications"). Notifications is a Conformist: it takes Ticketing's
// events as they are.
package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/application"
	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
)

type sendTickets interface {
	Handle(context.Context, application.SendTickets) error
}

type sendRefund interface {
	Handle(context.Context, application.SendRefund) error
}

// TicketingConsumer translates Ticketing's events into Notifications'
// commands.
type TicketingConsumer struct {
	tickets sendTickets
	refund  sendRefund
}

// NewTicketingConsumer wires the consumer.
func NewTicketingConsumer(tickets sendTickets, refund sendRefund) *TicketingConsumer {
	return &TicketingConsumer{tickets: tickets, refund: refund}
}

// Handle is a kafka.Handler. Sold-out facts are none of Notifications'
// business.
func (c *TicketingConsumer) Handle(ctx context.Context, env kafka.Envelope) error {
	switch env.EventType {
	case contracts.TypeTicketsIssuedV1:
		var e contracts.TicketsIssuedV1
		if err := decode(env, &e); err != nil {
			return err
		}
		cmd := application.SendTickets{OrderID: e.OrderID, ShowID: e.ShowID, ContactEmail: e.ContactEmail}
		for _, t := range e.Tickets {
			cmd.Tickets = append(cmd.Tickets, application.Ticket{Seat: t.Seat, Code: t.Code})
		}
		return wrap(env, c.tickets.Handle(ctx, cmd))
	case contracts.TypeOrderRefundedV1:
		var e contracts.OrderRefundedV1
		if err := decode(env, &e); err != nil {
			return err
		}
		return wrap(env, c.refund.Handle(ctx, application.SendRefund{
			OrderID: e.OrderID, ContactEmail: e.ContactEmail, Amount: e.Amount, Currency: e.Currency,
		}))
	default:
		return nil
	}
}

func decode(env kafka.Envelope, v any) error {
	if err := json.Unmarshal(env.Payload, v); err != nil {
		return fmt.Errorf("decode %s: %w", env.EventType, err)
	}
	return nil
}

func wrap(env kafka.Envelope, err error) error {
	if err != nil {
		return fmt.Errorf("%s: %w", env.EventType, err)
	}
	return nil
}
