//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

type outboxRow struct {
	Type, Key string
	Published bool
}

func outboxRows(t *testing.T, pool *pgxpool.Pool, venueID domain.VenueID) []outboxRow {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT event_type, msg_key, published_at IS NOT NULL FROM venue.outbox WHERE msg_key = $1 ORDER BY id`, venueID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.Type, &r.Key, &r.Published); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

func activeVenue(t *testing.T) *domain.Venue {
	t.Helper()
	addr, _ := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	v, err := domain.RegisterVenue(domain.NewVenueID(uuid.New()), "Coliseu", addr, at)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := domain.NewSectionCode("FLOOR")
	floor, _ := domain.NewGeneralAdmissionSection(code, "Floor", 500)
	if err := v.AddSection(floor, at); err != nil {
		t.Fatal(err)
	}
	if err := v.Activate(at); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSave_WritesTheOutboxInTheSameTransaction(t *testing.T) {
	pool := pgtest.New(t)
	repo := postgres.NewVenueRepository(pool)
	venue := activeVenue(t)

	if err := repo.Save(context.Background(), venue); err != nil {
		t.Fatal(err)
	}

	got := outboxRows(t, pool, venue.ID())
	want := []outboxRow{{Type: "venue.activated.v1", Key: venue.ID().String(), Published: false}}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("outbox = %+v, want %+v", got, want)
	}
}

func TestSave_AFailedSaveLeavesNoOutboxRow(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	repo := postgres.NewVenueRepository(pool)
	venue := activeVenue(t)
	if err := repo.Save(ctx, venue); err != nil {
		t.Fatal(err)
	}
	first, _ := repo.Get(ctx, venue.ID())
	second, _ := repo.Get(ctx, venue.ID())
	if err := first.Retire(at); err != nil {
		t.Fatal(err)
	}
	if err := second.Retire(at); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, first); err != nil {
		t.Fatal(err)
	}

	err := repo.Save(ctx, second)

	if !errors.Is(err, application.ErrConcurrentModification) {
		t.Fatalf("Save() error = %v, want ErrConcurrentModification", err)
	}
	retired := 0
	for _, r := range outboxRows(t, pool, venue.ID()) {
		if r.Type == "venue.retired.v1" {
			retired++
		}
	}
	if retired != 1 {
		t.Errorf("%d venue.retired.v1 rows, want 1: the losing save must not leave one", retired)
	}
}
