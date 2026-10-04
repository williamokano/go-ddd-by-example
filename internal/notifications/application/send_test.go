package application_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/williamokano/go-ddd-by-example/internal/notifications/application"
)

var update = flag.Bool("update", false, "rewrite the golden emails")

// fakeTx runs fn directly: the memory inbox has nothing to roll back.
type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

// memoryInbox is an Inbox fake.
type memoryInbox map[string]bool

func (m memoryInbox) Claim(_ context.Context, eventID string) (bool, error) {
	first := !m[eventID]
	m[eventID] = true
	return first, nil
}

// outbox is an EmailSender spy.
type outbox struct{ sent []application.Email }

func (o *outbox) Send(_ context.Context, e application.Email) error {
	o.sent = append(o.sent, e)
	return nil
}

func golden(t *testing.T, name string, got application.Email) {
	t.Helper()
	text := "To: " + got.To + "\nSubject: " + got.Subject + "\n\n" + got.Body
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(string(want), text); diff != "" {
		t.Errorf("email (-golden +got):\n%s", diff)
	}
}

func TestSendTickets_WritesToTheContactEmailTheEventCarries(t *testing.T) {
	sender := &outbox{}
	err := application.NewSendTicketsHandler(fakeTx{}, memoryInbox{}, sender).Handle(context.Background(), application.SendTickets{
		EventID:      "evt-1",
		OrderID:      "0199a0e0-0000-7000-8000-000000000001",
		ShowID:       "0199a0e0-0000-7000-8000-0000000000aa",
		ContactEmail: "ana@example.com",
		Tickets: []application.Ticket{
			{Seat: "ORCH/A/1", Code: "TCK-AAAA-1111"},
			{Seat: "ORCH/A/2", Code: "TCK-BBBB-2222"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d emails, want 1", len(sender.sent))
	}
	golden(t, "tickets_issued.golden", sender.sent[0])
}

func TestSendRefund_SaysHowMuchWentBack(t *testing.T) {
	sender := &outbox{}
	err := application.NewSendRefundHandler(fakeTx{}, memoryInbox{}, sender).Handle(context.Background(), application.SendRefund{
		EventID:      "evt-2",
		OrderID:      "0199a0e0-0000-7000-8000-000000000001",
		ContactEmail: "ana@example.com",
		Amount:       9000,
		Currency:     "EUR",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sent %d emails, want 1", len(sender.sent))
	}
	golden(t, "order_refunded.golden", sender.sent[0])
}

// Kafka delivers at least once: the same tickets_issued.v1 may arrive twice.
func TestSendTickets_ADuplicateEventDoesNotSendTwice(t *testing.T) {
	sender := &outbox{}
	h := application.NewSendTicketsHandler(fakeTx{}, memoryInbox{}, sender)
	cmd := application.SendTickets{EventID: "evt-1", OrderID: "o1", ContactEmail: "ana@example.com"}

	for range 2 {
		if err := h.Handle(context.Background(), cmd); err != nil {
			t.Fatal(err)
		}
	}

	if len(sender.sent) != 1 {
		t.Errorf("sent %d emails for one event, want 1", len(sender.sent))
	}
}
