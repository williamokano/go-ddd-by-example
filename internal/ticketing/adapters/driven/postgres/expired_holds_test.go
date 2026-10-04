//go:build integration

package postgres_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/inventoryrepotest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestExpiredHolds(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	repo := postgres.NewInventoryRepository(pool)
	inv := inventoryrepotest.Open(t)
	ref, _ := domain.ParseSeatRef("ORCH/A/1")
	if err := inv.Hold(domain.NewHoldID(uuid.New()), domain.NewCustomerID(uuid.New()), []domain.SeatRef{ref}, now, 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, inv); err != nil {
		t.Fatal(err)
	}
	finder := postgres.NewExpiredHolds(pool)

	before, _ := finder.ShowsWithExpiredHolds(ctx, now.Add(9*time.Minute))
	after, err := finder.ShowsWithExpiredHolds(ctx, now.Add(10*time.Minute))

	if err != nil || slices.Contains(before, inv.ShowID()) || !slices.Contains(after, inv.ShowID()) {
		t.Errorf("before = %v, after = %v, %v", before, after, err)
	}
}
