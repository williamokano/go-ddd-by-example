package postgres_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/sagamsg"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/contracts"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestToOutboxMessages_SagaSteps(t *testing.T) {
	at := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	show, hold, order := domain.NewShowID(uuid.New()), domain.NewHoldID(uuid.New()), domain.NewOrderID(uuid.New())
	seat, _ := domain.ParseSeatRef("ORCH/A/1")

	msgs, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.SeatsHeld{ShowID: show, At: at}, // internal fact: not a message
		domain.OrderPaid{OrderID: order, ShowID: show, Section: "ORCH", HoldID: hold, At: at},
		domain.SeatsSold{ShowID: show, HoldID: hold, OrderID: order, Seats: []domain.SeatRef{seat}, At: at},
		domain.HoldConfirmationFailed{ShowID: show, HoldID: hold, OrderID: order, Reason: "expired", At: at},
	}, uuid.New)

	if err != nil || len(msgs) != 3 {
		t.Fatalf("got %d messages, %v; want 3", len(msgs), err)
	}
	for i, want := range []string{sagamsg.TypeOrderPaid, sagamsg.TypeSeatsSold, sagamsg.TypeHoldConfirmationFailed} {
		if msgs[i].Type != want || msgs[i].Topic != sagamsg.Topic || msgs[i].Key != order.String() {
			t.Errorf("message %d = %s on %s keyed %s; want %s on %s keyed by the order", i, msgs[i].Type, msgs[i].Topic, msgs[i].Key, want, sagamsg.Topic)
		}
	}
	var paid sagamsg.OrderPaid
	if err := json.Unmarshal(msgs[0].Payload, &paid); err != nil || paid.Section != "ORCH" {
		t.Errorf("order paid payload = %+v, %v; want the section", paid, err)
	}
	var sold sagamsg.SeatsSold
	if err := json.Unmarshal(msgs[1].Payload, &sold); err != nil || len(sold.Seats) != 1 || sold.Seats[0] != "ORCH/A/1" {
		t.Errorf("seats sold payload = %+v, %v", sold, err)
	}
}

func TestToOutboxMessages_PublishedLanguage(t *testing.T) {
	at := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	show, order := domain.NewShowID(uuid.New()), domain.NewOrderID(uuid.New())
	email, _ := domain.NewContactEmail("ana@example.com")
	seat, _ := domain.ParseSeatRef("ORCH/A/1")
	eur, _ := sharedkernel.NewCurrency("EUR")
	total, _ := sharedkernel.NewMoney(9000, eur)

	msgs, err := postgres.ToOutboxMessages([]sharedkernel.DomainEvent{
		domain.InventorySoldOut{ShowID: show, At: at},
		domain.OrderFulfilled{OrderID: order, ShowID: show, ContactEmail: email,
			Tickets: []domain.IssuedTicket{{Seat: seat, Code: domain.TicketCodeFor(domain.NewTicketID(uuid.New()))}}, At: at},
		domain.OrderRefunded{OrderID: order, ShowID: show, Section: "ORCH", ContactEmail: email, Total: total, At: at},
		domain.InventoryClosed{ShowID: show, At: at},
		domain.SectionSoldOut{ShowID: show, Section: "ORCH", At: at},
		domain.SectionBackOnSale{ShowID: show, Section: "ORCH", At: at},
	}, uuid.New)

	// OrderRefunded is also a saga step (9.5): its seats may go back on sale.
	if err != nil || len(msgs) != 7 {
		t.Fatalf("got %d messages, %v", len(msgs), err)
	}
	want := []struct{ topic, typ, key string }{
		{contracts.Topic, contracts.TypeInventorySoldOutV1, show.String()},
		{contracts.Topic, contracts.TypeTicketsIssuedV1, order.String()},
		{contracts.Topic, contracts.TypeOrderRefundedV1, order.String()},
		{sagamsg.Topic, sagamsg.TypeOrderRefunded, order.String()},
		{sagamsg.Topic, sagamsg.TypeInventoryClosed, show.String()},
		{sagamsg.Topic, sagamsg.TypeSectionSoldOut, show.String()},
		{contracts.Topic, contracts.TypeInventoryAvailableAgainV1, show.String()},
	}
	for i, w := range want {
		if msgs[i].Topic != w.topic || msgs[i].Type != w.typ || msgs[i].Key != w.key {
			t.Errorf("message %d = %s/%s/%s, want %s/%s/%s", i, msgs[i].Topic, msgs[i].Type, msgs[i].Key, w.topic, w.typ, w.key)
		}
	}
	var issued contracts.TicketsIssuedV1
	if err := json.Unmarshal(msgs[1].Payload, &issued); err != nil || issued.ContactEmail != "ana@example.com" || len(issued.Tickets) != 1 {
		t.Errorf("tickets issued = %+v, %v", issued, err)
	}
}
