package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// OpenInventory is Ticketing's own command for "a show went on sale": the
// consumer translates show.published.v1 into it.
type OpenInventory struct {
	ShowID   string
	StartsAt time.Time
	Sections []SectionSpec
}

// SectionSpec is one priced section of the published layout.
type SectionSpec struct {
	Code     string
	Kind     string
	Rows     []RowSpec
	Capacity int
	Price    int64 // minor units
	Currency string
}

// RowSpec is one row of a seated section.
type RowSpec struct {
	Label      string
	Seats      int
	Accessible []int // seat numbers with step-free access
}

// OpenInventoryHandler is the TKT-1 policy: one inventory per section of a
// published show (ADR-013).
type OpenInventoryHandler struct {
	inventories InventoryRepository
	clock       Clock
}

// NewOpenInventoryHandler wires the use case to its ports.
func NewOpenInventoryHandler(inventories InventoryRepository, clock Clock) *OpenInventoryHandler {
	return &OpenInventoryHandler{inventories: inventories, clock: clock}
}

// Handle opens every section once, each in its own transaction. A
// redelivered show.published.v1 opens only the sections a crash left
// unopened, and otherwise does nothing (idempotent, TKT-1).
func (h *OpenInventoryHandler) Handle(ctx context.Context, cmd OpenInventory) error {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return fmt.Errorf("open inventory: %w", err)
	}
	existing, err := h.inventories.ListByShow(ctx, showID)
	if err != nil {
		return fmt.Errorf("open inventory: %w", err)
	}
	opened := map[string]bool{}
	for _, inv := range existing {
		opened[inv.Section()] = true
	}
	layout, err := newLayout(cmd.Sections)
	if err != nil {
		return fmt.Errorf("open inventory: %w", err)
	}
	sections, err := domain.OpenInventory(showID, layout, cmd.StartsAt, h.clock.Now())
	if err != nil {
		return fmt.Errorf("open inventory: %w", err)
	}
	for _, inv := range sections {
		if opened[inv.Section()] {
			continue
		}
		if err := h.inventories.Save(ctx, inv); err != nil && !errors.Is(err, ErrConcurrentModification) {
			return fmt.Errorf("open inventory: section %s: %w", inv.Section(), err)
		} // a conflict means a duplicate delivery opened it first
	}
	return nil
}

func newLayout(specs []SectionSpec) (domain.InventoryLayout, error) {
	var layout domain.InventoryLayout
	for _, s := range specs {
		currency, err := sharedkernel.NewCurrency(s.Currency)
		if err != nil {
			return domain.InventoryLayout{}, err
		}
		price, err := sharedkernel.NewMoney(s.Price, currency)
		if err != nil {
			return domain.InventoryLayout{}, err
		}
		section := domain.InventorySection{Code: s.Code, Kind: s.Kind, Capacity: s.Capacity, Price: price}
		for _, r := range s.Rows {
			section.Rows = append(section.Rows, domain.InventoryRow{Label: r.Label, Seats: r.Seats, Accessible: r.Accessible})
		}
		layout.Sections = append(layout.Sections, section)
	}
	return layout, nil
}

// HoldSeats is the command to hold seats for a customer.
type HoldSeats struct {
	ShowID     string
	CustomerID string
	Seats      []string
}

// HoldResult says which hold was created and until when it lasts.
type HoldResult struct {
	HoldID    domain.HoldID
	ExpiresAt time.Time
}

// HoldSeatsHandler is the HoldSeats use case.
type HoldSeatsHandler struct {
	inventories InventoryRepository
	ids         IDGenerator
	clock       Clock
	ttl         time.Duration
}

// NewHoldSeatsHandler wires the use case; ttl is how long holds last (TKT-4).
func NewHoldSeatsHandler(inventories InventoryRepository, ids IDGenerator, clock Clock, ttl time.Duration) *HoldSeatsHandler {
	return &HoldSeatsHandler{inventories: inventories, ids: ids, clock: clock, ttl: ttl}
}

// Handle holds the seats in their section's inventory. Holds race only with
// holds in the same section (ADR-013); a lost race is retried a few times
// before the customer gets a conflict.
func (h *HoldSeatsHandler) Handle(ctx context.Context, cmd HoldSeats) (HoldResult, error) {
	showID, err := domain.ParseShowID(cmd.ShowID)
	if err != nil {
		return HoldResult{}, fmt.Errorf("hold seats: %w", err)
	}
	customer, err := domain.ParseCustomerID(cmd.CustomerID)
	if err != nil {
		return HoldResult{}, fmt.Errorf("hold seats: %w", err)
	}
	seats := make([]domain.SeatRef, 0, len(cmd.Seats))
	for _, raw := range cmd.Seats {
		ref, err := domain.ParseSeatRef(raw)
		if err != nil {
			return HoldResult{}, fmt.Errorf("hold seats: %w", err)
		}
		seats = append(seats, ref)
	}
	if len(seats) == 0 {
		return HoldResult{}, fmt.Errorf("hold seats: %w: no seats", domain.ErrInvalidHoldSize)
	}
	section := seats[0].Section() // the section checks the others
	id := h.ids.NewHoldID()
	var result HoldResult
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		inv, err := h.inventories.Get(ctx, showID, section)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		now := h.clock.Now()
		if err := inv.Hold(id, customer, seats, now, h.ttl); err != nil {
			return err
		}
		if err := h.inventories.Save(ctx, inv); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		result = HoldResult{HoldID: id, ExpiresAt: now.Add(h.ttl)}
		return nil
	})
	if err != nil {
		return HoldResult{}, fmt.Errorf("hold seats: %w", err)
	}
	return result, nil
}

// ReleaseHold is the command for a customer giving their seats back.
type ReleaseHold struct {
	HoldID     string
	CustomerID string
}

// ReleaseHoldHandler is the ReleaseHold use case.
type ReleaseHoldHandler struct {
	inventories InventoryRepository
	clock       Clock
}

// NewReleaseHoldHandler wires the use case.
func NewReleaseHoldHandler(inventories InventoryRepository, clock Clock) *ReleaseHoldHandler {
	return &ReleaseHoldHandler{inventories: inventories, clock: clock}
}

// Handle releases the hold, if it is the customer's.
func (h *ReleaseHoldHandler) Handle(ctx context.Context, cmd ReleaseHold) error {
	holdID, err := domain.ParseHoldID(cmd.HoldID)
	if err != nil {
		return fmt.Errorf("release hold: %w", err)
	}
	customer, err := domain.ParseCustomerID(cmd.CustomerID)
	if err != nil {
		return fmt.Errorf("release hold: %w", err)
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		inv, err := h.inventories.GetByHold(ctx, holdID)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		if err := inv.ReleaseHold(holdID, customer, h.clock.Now()); err != nil {
			return err
		}
		if err := h.inventories.Save(ctx, inv); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("release hold: %w", err)
	}
	return nil
}
