//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/clock"
	"github.com/williamokano/go-ddd-by-example/internal/platform/idgen"
	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/ids"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

var now = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

// counting counts lost races (saves rejected as stale).
type counting struct {
	application.InventoryRepository
	conflicts atomic.Int64
}

func (c *counting) Save(ctx context.Context, inv *domain.SectionInventory) error {
	err := c.InventoryRepository.Save(ctx, inv)
	if errors.Is(err, application.ErrConcurrentModification) {
		c.conflicts.Add(1)
	}
	return err
}

// openShow opens a show of GA sections FLOOR0, FLOOR1, ... of places each.
func openShow(t *testing.T, repo application.InventoryRepository, sections, places int) string {
	t.Helper()
	showID := uuid.NewString()
	open := application.NewOpenInventoryHandler(repo, clock.NewFixed(now))
	cmd := application.OpenInventory{ShowID: showID, StartsAt: now.Add(30 * 24 * time.Hour)}
	for i := range sections {
		cmd.Sections = append(cmd.Sections, application.SectionSpec{
			Code: fmt.Sprintf("FLOOR%d", i), Kind: "ga", Capacity: places, Price: 2500, Currency: "EUR",
		})
	}
	err := open.Handle(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	return showID
}

// S6 — 1 available seat, 50 concurrent holds: exactly 1 wins (TKT-5).
func TestS6_RaceForTheLastSeat(t *testing.T) {
	repo := postgres.NewInventoryRepository(pgtest.New(t))
	showID := openShow(t, repo, 1, 1)
	hold := application.NewHoldSeatsHandler(repo, ids.New(idgen.UUIDv7{}), clock.NewFixed(now), 10*time.Minute)

	errs := make([]error, 50)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Go(func() {
			_, errs[i] = hold.Handle(context.Background(), application.HoldSeats{
				ShowID: showID, CustomerID: uuid.NewString(), Seats: []string{"FLOOR0/GA/0001"},
			})
		})
	}
	wg.Wait()

	var wins, unavailable, exhausted int
	for _, err := range errs {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, domain.ErrSeatUnavailable):
			unavailable++
		case errors.Is(err, application.ErrConcurrentModification):
			exhausted++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if wins != 1 {
		t.Fatalf("%d holds won the last seat, want exactly 1 (TKT-5)", wins)
	}
	t.Logf("1 win, %d seat unavailable, %d gave up after %d conflicts", unavailable, exhausted, 3)
}

// The cost of the aggregate's size: 200 customers hold 200 different seats at
// once. With one section (what ADR-005's one inventory per show amounts to)
// they all race for one version; with ten sections (ADR-013), for ten.
func TestHoldContention_Measured(t *testing.T) {
	if testing.Short() {
		t.Skip("measurement")
	}
	for _, sections := range []int{1, 10} {
		t.Run(fmt.Sprintf("%d sections", sections), func(t *testing.T) {
			base := postgres.NewInventoryRepository(pgtest.New(t))
			repo := &counting{InventoryRepository: base}
			showID := openShow(t, base, sections, 1000/sections)
			hold := application.NewHoldSeatsHandler(repo, ids.New(idgen.UUIDv7{}), clock.NewFixed(now), 10*time.Minute)

			latencies := make([]time.Duration, 200)
			errs := make([]error, 200)
			var wg sync.WaitGroup
			for i := range latencies {
				wg.Go(func() {
					seat := fmt.Sprintf("FLOOR%d/GA/%04d", i%sections, i/sections+1)
					start := time.Now()
					_, errs[i] = hold.Handle(context.Background(), application.HoldSeats{
						ShowID: showID, CustomerID: uuid.NewString(), Seats: []string{seat},
					})
					latencies[i] = time.Since(start)
				})
			}
			wg.Wait()

			held := 0
			for _, err := range errs {
				if err == nil {
					held++
				}
			}
			slices.Sort(latencies)
			t.Logf("%d section(s) of %d places: %d/200 holds succeeded, %d conflicts retried, p50 %s, p99 %s",
				sections, 1000/sections, held, repo.conflicts.Load(),
				latencies[len(latencies)/2].Round(time.Millisecond), latencies[len(latencies)*99/100].Round(time.Millisecond))
		})
	}
}
