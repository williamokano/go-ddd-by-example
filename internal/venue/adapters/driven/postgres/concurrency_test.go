//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// ADR-011 under real contention: two writers load the same draft venue and
// each add a different section. Exactly one save may win.
func TestVenueRepository_ConcurrentAddSection(t *testing.T) {
	ctx := context.Background()
	repo := postgres.NewVenueRepository(pgtest.New(t))
	now := time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)
	addr, _ := domain.NewAddress("Rua Portas de Santo Antão 96", "Lisboa", "PT")
	venue, err := domain.RegisterVenue(domain.NewVenueID(uuid.New()), "Coliseu", addr, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, venue); err != nil {
		t.Fatal(err)
	}

	codes := []string{"LEFT", "RIGHT"}
	errs := make([]error, len(codes))
	var allLoaded, done sync.WaitGroup
	allLoaded.Add(len(codes))
	for i, raw := range codes {
		done.Go(func() {
			// Both writers load the same version before either saves;
			// otherwise the second would simply load the first one's result.
			v, err := repo.Get(ctx, venue.ID())
			allLoaded.Done()
			allLoaded.Wait()
			if err != nil {
				errs[i] = err
				return
			}
			code, _ := domain.NewSectionCode(raw)
			section, _ := domain.NewGeneralAdmissionSection(code, raw, 100)
			if err := v.AddSection(section, now); err != nil {
				errs[i] = err
				return
			}
			errs[i] = repo.Save(ctx, v)
		})
	}
	done.Wait()

	conflicts := 0
	for _, err := range errs {
		if errors.Is(err, application.ErrConcurrentModification) {
			conflicts++
		} else if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if conflicts != 1 {
		t.Errorf("conflicts = %d, want exactly 1", conflicts)
	}
	stored, err := repo.Get(ctx, venue.ID())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(stored.Sections()); got != 1 {
		t.Errorf("stored venue has %d sections, want exactly 1 (the winner's)", got)
	}
}
