// Package application is all of Notifications: a generic subdomain, so a
// transaction script per event and a port to send through. No domain
// package, no aggregates, no repository (Chapter 1: Conformist).
package application

import (
	"context"
	"fmt"
	"strings"
)

// Email is what gets sent.
type Email struct {
	To      string
	Subject string
	Body    string
}

// EmailSender is the driven port to whatever delivers email.
type EmailSender interface {
	Send(ctx context.Context, e Email) error
}

// TxManager runs fn in one transaction (ADR-012): the inbox claim commits
// only if the email went out.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Inbox remembers processed events. Claim reports whether this is the event's
// first delivery; it must run inside WithinTx.
type Inbox interface {
	Claim(ctx context.Context, eventID string) (bool, error)
}

// script is what both handlers share: claim the event, then send, in one
// transaction. Kafka delivers at least once; the inbox makes the email go
// out effectively once (8.4).
type script struct {
	tx     TxManager
	inbox  Inbox
	sender EmailSender
}

func (s script) sendOnce(ctx context.Context, eventID string, e Email) error {
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		first, err := s.inbox.Claim(ctx, eventID)
		if err != nil || !first {
			return err //nolint:wrapcheck // wrapped below, once
		}
		return s.sender.Send(ctx, e) //nolint:wrapcheck // wrapped below, once
	})
	if err != nil {
		return fmt.Errorf("send %q to %s: %w", e.Subject, e.To, err)
	}
	return nil
}

// Ticket is one issued ticket, as the customer needs it.
type Ticket struct {
	Seat string
	Code string
}

// SendTickets is the command to email an order's tickets. Everything comes
// from Ticketing's event: Notifications never looks a customer up.
type SendTickets struct {
	EventID      string // the inbox key
	OrderID      string
	ShowID       string
	ContactEmail string
	Tickets      []Ticket
}

// SendTicketsHandler formats and sends the tickets email.
type SendTicketsHandler struct{ script }

// NewSendTicketsHandler wires the script to its ports.
func NewSendTicketsHandler(tx TxManager, inbox Inbox, sender EmailSender) *SendTicketsHandler {
	return &SendTicketsHandler{script{tx: tx, inbox: inbox, sender: sender}}
}

// Handle sends one email listing every ticket.
func (h *SendTicketsHandler) Handle(ctx context.Context, cmd SendTickets) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Your tickets for order %s (show %s):\n\n", cmd.OrderID, cmd.ShowID)
	for _, t := range cmd.Tickets {
		fmt.Fprintf(&b, "  %-14s %s\n", t.Seat, t.Code)
	}
	b.WriteString("\nShow the code at the door. Enjoy the show!\n")
	return h.sendOnce(ctx, cmd.EventID, Email{
		To:      cmd.ContactEmail,
		Subject: fmt.Sprintf("Your %d ticket(s) for order %s", len(cmd.Tickets), cmd.OrderID),
		Body:    b.String(),
	})
}

// SendRefund is the command to tell a customer their money went back.
type SendRefund struct {
	EventID      string // the inbox key
	OrderID      string
	ContactEmail string
	Amount       int64 // minor units
	Currency     string
}

// SendRefundHandler formats and sends the refund email.
type SendRefundHandler struct{ script }

// NewSendRefundHandler wires the script to its ports.
func NewSendRefundHandler(tx TxManager, inbox Inbox, sender EmailSender) *SendRefundHandler {
	return &SendRefundHandler{script{tx: tx, inbox: inbox, sender: sender}}
}

// Handle sends the refund email.
func (h *SendRefundHandler) Handle(ctx context.Context, cmd SendRefund) error {
	return h.sendOnce(ctx, cmd.EventID, Email{
		To:      cmd.ContactEmail,
		Subject: "Your order " + cmd.OrderID + " was refunded",
		Body: fmt.Sprintf("We refunded %s %d.%02d for order %s.\nAny tickets of this order are no longer valid.\n",
			cmd.Currency, cmd.Amount/100, cmd.Amount%100, cmd.OrderID),
	})
}
