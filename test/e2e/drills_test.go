//go:build e2e && drills

// Failure drills (8.5): each one breaks a piece of the running stack on
// purpose and checks the system converges once it is back. They stop and
// start Compose services, so they run apart from the e2e suite:
//
//	make drills
package e2e_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/williamokano/go-ddd-by-example/internal/platform/kafka"
	showcontracts "github.com/williamokano/go-ddd-by-example/internal/show/contracts"
)

// compose runs a docker compose command. STAGEHAND_COMPOSE overrides the
// base command, e.g. to add -f files.
func compose(t *testing.T, args ...string) {
	t.Helper()
	base := strings.Fields(os.Getenv("STAGEHAND_COMPOSE"))
	if len(base) == 0 {
		base = []string{"docker", "compose"}
	}
	cmd := exec.Command(base[0], append(base[1:], args...)...)
	cmd.Dir = "../.." // where docker-compose.yml is
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", strings.Join(cmd.Args, " "), err, out)
	}
}

// unpublished counts the purchase's rows still waiting in ticketing.outbox.
func unpublished(t *testing.T, correlation string) int {
	t.Helper()
	conn, err := pgx.Connect(t.Context(), databaseURL())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	var n int
	err = conn.QueryRow(t.Context(),
		`SELECT count(*) FROM ticketing.outbox WHERE correlation_id = $1 AND published_at IS NULL`, correlation).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Drill 1: Kafka goes down in the middle of S1. The HTTP part completes, the
// outbox grows; Kafka comes back and the saga finishes. (Outbox.)
func TestDrill_KafkaDownMidPurchase(t *testing.T) {
	c := newClient(t)
	show := c.publishedShow()
	customer := uuid.NewString()
	h := c.hold(show, customer, "ORCH/A/1")

	compose(t, "stop", "kafka")
	t.Cleanup(func() { compose(t, "up", "-d", "--wait", "kafka") })
	flow := "drill-" + uuid.NewString()
	order := c.checkout(h, customer, map[string]string{"X-Correlation-ID": flow})

	if got := c.order(order).Status; got != "paid" {
		t.Errorf("with Kafka down the order is %s, want paid (the saga waits)", got)
	}
	if n := unpublished(t, flow); n == 0 {
		t.Error("nothing waits in the outbox")
	}

	compose(t, "up", "-d", "--wait", "kafka")
	eventually(t, 90*time.Second, "the saga finishes once Kafka is back", func() bool {
		return c.order(order).Status == "fulfilled"
	})
	eventually(t, 10*time.Second, "the outbox drains", func() bool {
		return unpublished(t, flow) == 0
	})
}

// Drill 2: the app is killed while purchases are in flight. On restart the
// relay republishes what it had not marked, and every handler absorbs the
// duplicates. (Outbox + idempotency.)
func TestDrill_AppKilledWhilePublishing(t *testing.T) {
	c := newClient(t)
	shows := []string{c.publishedShow(), c.publishedShow(), c.publishedShow()}
	var orders []string
	for _, show := range shows {
		customer := uuid.NewString()
		orders = append(orders, c.checkout(c.hold(show, customer, "ORCH/A/1", "ORCH/A/2"), customer, nil))
	}

	compose(t, "kill", "app")
	compose(t, "up", "-d", "--wait", "--no-deps", "app")

	for _, id := range orders {
		var got orderJSON
		eventually(t, 60*time.Second, "order "+id+" is fulfilled", func() bool {
			got = c.order(id)
			return got.Status == "fulfilled"
		})
		if len(got.Tickets) != 2 {
			t.Errorf("order %s has %d tickets, want 2: a duplicate was not absorbed", id, len(got.Tickets))
		}
	}
}

// Drill 3: a malformed message on show.events goes to show.events.dlq and
// consumption continues. (DLQ.)
func TestDrill_MalformedMessageGoesToTheDLQ(t *testing.T) {
	c := newClient(t)
	marker := "drill-" + uuid.NewString()
	client, err := kgo.NewClient(kgo.SeedBrokers(kafkaBroker()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	err = client.ProduceSync(t.Context(), &kgo.Record{Topic: "show.events", Key: []byte(marker), Value: []byte("{not json")}).FirstErr()
	if err != nil {
		t.Fatal(err)
	}

	dlq, err := kgo.NewClient(kgo.SeedBrokers(kafkaBroker()), kgo.ConsumeTopics("show.events.dlq"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	if err != nil {
		t.Fatal(err)
	}
	defer dlq.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	found := false
	for !found && ctx.Err() == nil {
		dlq.PollFetches(ctx).EachRecord(func(r *kgo.Record) {
			found = found || string(r.Key) == marker
		})
	}
	if !found {
		t.Fatal("the malformed message never reached show.events.dlq")
	}

	show := c.publishedShow()
	c.hold(show, uuid.NewString(), "ORCH/A/1") // Ticketing still consumes show.events
}

// Drill 4: Postgres goes down. HTTP answers 500, consumers back off without
// losing messages, and everything converges when it is back. (Offsets
// committed only after handling.)
func TestDrill_PostgresDown(t *testing.T) {
	c := newClient(t)
	compose(t, "stop", "postgres")
	t.Cleanup(func() { compose(t, "up", "-d", "--wait", "postgres") })
	if r := c.do(http.MethodGet, "/venues", nil); r.Status != http.StatusInternalServerError {
		t.Errorf("GET /venues with Postgres down: %d, want 500", r.Status)
	}

	// A show is published while Ticketing cannot write: it must retry until
	// Postgres answers, not dead-letter the event.
	show := publishByHand(t)
	time.Sleep(10 * time.Second)
	compose(t, "up", "-d", "--wait", "postgres")

	c.hold(show, uuid.NewString(), "ORCH/A/1") // the inventory opened: nothing was lost
}

// publishByHand puts a show.published.v2 straight on show.events, as Show's
// relay would, and returns the show's id.
func publishByHand(t *testing.T) string {
	t.Helper()
	show := uuid.NewString()
	start := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Minute)
	payload, err := json.Marshal(showcontracts.ShowPublishedV2{
		ShowID: show, VenueID: uuid.NewString(), Title: "Drill Night",
		DoorsOpen: start.Add(-time.Hour), StartsAt: start, EndsAt: start.Add(2 * time.Hour),
		Sections: []showcontracts.SectionV2{{
			Code: "ORCH", Kind: showcontracts.KindSeated, Seats: []showcontracts.SeatV2{{Row: "A", Number: 1}, {Row: "A", Number: 2}},
			Price: showcontracts.PriceV1{Amount: 4500, Currency: "EUR"},
		}},
		PublishedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err := json.Marshal(kafka.Envelope{
		EventID: uuid.NewString(), EventType: showcontracts.TypeShowPublishedV2, OccurredAt: time.Now().UTC(),
		AggregateID: show, Payload: payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	client, err := kgo.NewClient(kgo.SeedBrokers(kafkaBroker()))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.ProduceSync(t.Context(), &kgo.Record{Topic: showcontracts.Topic, Key: []byte(show), Value: value}).FirstErr(); err != nil {
		t.Fatal(err)
	}
	return show
}

func kafkaBroker() string {
	if b := os.Getenv("STAGEHAND_KAFKA"); b != "" {
		return b
	}
	return "localhost:9092"
}
