package application_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
)

func TestExpireHolds(t *testing.T) {
	f := newFixture(t)
	show := f.openShow(t)
	f.holdSeats(t, show, uuid.NewString(), "ORCH/A/1")
	expire := application.NewExpireHoldsHandler(f.inventories, memory.NewExpiredHolds(f.inventories), f.clock)

	f.clock.Advance(9 * time.Minute)
	if err := expire.Handle(f.ctx); err != nil {
		t.Fatal(err)
	}
	if f.seatStates(t, show)["ORCH/A/1"] != "held" {
		t.Fatal("expired a hold before its time")
	}

	f.clock.Advance(2 * time.Minute) // 11 minutes after the hold
	if err := expire.Handle(f.ctx); err != nil {
		t.Fatal(err)
	}

	if f.seatStates(t, show)["ORCH/A/1"] != "available" {
		t.Error("the seat is still held 11 minutes later (TKT-4)")
	}
}
