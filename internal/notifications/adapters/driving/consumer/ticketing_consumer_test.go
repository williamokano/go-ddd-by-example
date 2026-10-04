package consumer_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/adapters/driving/consumer"
	"github.com/williamokano/go-ddd-by-example/internal/notifications/application"
	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
)

type spy struct {
	tickets []application.SendTickets
	refunds []application.SendRefund
}

func (s *spy) Handle(_ context.Context, cmd application.SendTickets) error {
	s.tickets = append(s.tickets, cmd)
	return nil
}

type refundSpy struct{ s *spy }

func (r refundSpy) Handle(_ context.Context, cmd application.SendRefund) error {
	r.s.refunds = append(r.s.refunds, cmd)
	return nil
}

func envelope(t *testing.T, eventType string, payload any) kafka.Envelope {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return kafka.Envelope{EventID: "e1", EventType: eventType, Payload: b}
}

func TestTicketingConsumer_TranslatesTheTwoEventsNotificationsCaresAbout(t *testing.T) {
	s := &spy{}
	c := consumer.NewTicketingConsumer(s, refundSpy{s})
	ctx := context.Background()

	if err := c.Handle(ctx, envelope(t, contracts.TypeTicketsIssuedV1, contracts.TicketsIssuedV1{
		OrderID: "o1", ShowID: "s1", ContactEmail: "ana@example.com",
		Tickets: []contracts.TicketV1{{Seat: "ORCH/A/1", Code: "C1"}},
	})); err != nil {
		t.Fatal(err)
	}
	if err := c.Handle(ctx, envelope(t, contracts.TypeOrderRefundedV1, contracts.OrderRefundedV1{
		OrderID: "o1", ContactEmail: "ana@example.com", Amount: 9000, Currency: "EUR",
	})); err != nil {
		t.Fatal(err)
	}
	if err := c.Handle(ctx, envelope(t, contracts.TypeInventorySoldOutV1, contracts.InventorySoldOutV1{ShowID: "s1"})); err != nil {
		t.Fatal(err)
	}

	wantTickets := []application.SendTickets{{
		EventID: "e1", OrderID: "o1", ShowID: "s1", ContactEmail: "ana@example.com",
		Tickets: []application.Ticket{{Seat: "ORCH/A/1", Code: "C1"}},
	}}
	wantRefunds := []application.SendRefund{{EventID: "e1", OrderID: "o1", ContactEmail: "ana@example.com", Amount: 9000, Currency: "EUR"}}
	if diff := cmp.Diff(wantTickets, s.tickets); diff != "" {
		t.Errorf("tickets (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantRefunds, s.refunds); diff != "" {
		t.Errorf("refunds (-want +got):\n%s", diff)
	}
}

func TestTicketingConsumer_AMalformedPayloadIsAnError(t *testing.T) {
	c := consumer.NewTicketingConsumer(&spy{}, refundSpy{&spy{}})
	err := c.Handle(context.Background(), kafka.Envelope{EventType: contracts.TypeTicketsIssuedV1, Payload: []byte("{")})
	if err == nil {
		t.Error("err = nil, want a decode error (the runner sends it to the DLQ)")
	}
}
