//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/inventoryrepotest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

func TestShowSchedule(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	orch, _ := inventoryrepotest.Open(t)
	if err := postgres.NewInventoryRepository(pool).Save(ctx, orch); err != nil {
		t.Fatal(err)
	}
	schedule := postgres.NewShowSchedule(pool)

	got, err := schedule.StartsAt(ctx, orch.ShowID())
	if err != nil || !got.Equal(orch.StartsAt()) {
		t.Errorf("StartsAt() = %v, %v; want %v", got, err, orch.StartsAt())
	}
	if _, err := schedule.StartsAt(ctx, domain.NewShowID(uuid.New())); !errors.Is(err, application.ErrInventoryNotFound) {
		t.Errorf("error = %v, want %v", err, application.ErrInventoryNotFound)
	}
}
