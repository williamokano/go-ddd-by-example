package application

import (
	"context"
	"fmt"

	"github.com/williamokano/go-ddd-by-example/internal/sharedkernel"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// PriceShow sets a draft show's price list.
type PriceShow struct {
	ShowID     string
	PromoterID string
	Prices     []PriceSpec
}

// PriceSpec is one section's price, in minor units.
type PriceSpec struct {
	Section  string
	Amount   int64
	Currency string
}

// PublishShow puts a priced show on sale.
type PublishShow struct {
	ShowID     string
	PromoterID string
}

// CancelShow cancels a show, with a reason.
type CancelShow struct {
	ShowID     string
	PromoterID string
	Reason     string
}

// MarkShowSoldOut records Ticketing's fact that the show sold out. It acts
// for Ticketing, not for a promoter, so there is no promoter check.
type MarkShowSoldOut struct{ ShowID string }

// PriceShowHandler is the PriceShow use case.
type PriceShowHandler struct {
	shows   ShowRepository
	layouts VenueLayouts
	clock   Clock
}

// NewPriceShowHandler wires the use case to its ports.
func NewPriceShowHandler(shows ShowRepository, layouts VenueLayouts, clock Clock) *PriceShowHandler {
	return &PriceShowHandler{shows: shows, layouts: layouts, clock: clock}
}

// Handle prices the show.
func (h *PriceShowHandler) Handle(ctx context.Context, cmd PriceShow) error {
	prices, err := newPriceList(cmd.Prices)
	if err != nil {
		return fmt.Errorf("price show: %w", err)
	}
	return withShow(ctx, h.shows, "price show", cmd.ShowID, cmd.PromoterID, func(show *domain.Show) error {
		layout, err := h.layouts.Get(ctx, show.VenueID())
		if err != nil {
			return fmt.Errorf("venue layout: %w", err)
		}
		return show.Price(prices, layout, h.clock.Now())
	})
}

// PublishShowHandler is the PublishShow use case.
type PublishShowHandler struct {
	shows   ShowRepository
	layouts VenueLayouts
	clock   Clock
}

// NewPublishShowHandler wires the use case to its ports.
func NewPublishShowHandler(shows ShowRepository, layouts VenueLayouts, clock Clock) *PublishShowHandler {
	return &PublishShowHandler{shows: shows, layouts: layouts, clock: clock}
}

// Handle publishes the show with a snapshot of its venue's layout.
func (h *PublishShowHandler) Handle(ctx context.Context, cmd PublishShow) error {
	return withShow(ctx, h.shows, "publish show", cmd.ShowID, cmd.PromoterID, func(show *domain.Show) error {
		layout, err := h.layouts.Get(ctx, show.VenueID())
		if err != nil {
			return fmt.Errorf("venue layout: %w", err)
		}
		return show.Publish(layout, h.clock.Now())
	})
}

// CancelShowHandler is the CancelShow use case.
type CancelShowHandler struct {
	shows ShowRepository
	clock Clock
}

// NewCancelShowHandler wires the use case to its ports.
func NewCancelShowHandler(shows ShowRepository, clock Clock) *CancelShowHandler {
	return &CancelShowHandler{shows: shows, clock: clock}
}

// Handle cancels the show.
func (h *CancelShowHandler) Handle(ctx context.Context, cmd CancelShow) error {
	reason, err := domain.NewCancellationReason(cmd.Reason)
	if err != nil {
		return fmt.Errorf("cancel show: %w", err)
	}
	return withShow(ctx, h.shows, "cancel show", cmd.ShowID, cmd.PromoterID, func(show *domain.Show) error {
		return show.Cancel(reason, h.clock.Now())
	})
}

// MarkShowSoldOutHandler is the SHW-8 policy, driven by Ticketing's fact.
type MarkShowSoldOutHandler struct {
	shows ShowRepository
	clock Clock
}

// NewMarkShowSoldOutHandler wires the use case to its ports.
func NewMarkShowSoldOutHandler(shows ShowRepository, clock Clock) *MarkShowSoldOutHandler {
	return &MarkShowSoldOutHandler{shows: shows, clock: clock}
}

// Handle marks the show sold out (a no-op if that is old news).
func (h *MarkShowSoldOutHandler) Handle(ctx context.Context, cmd MarkShowSoldOut) error {
	return withShow(ctx, h.shows, "mark show sold out", cmd.ShowID, "", func(show *domain.Show) error {
		return show.MarkSoldOut(h.clock.Now())
	})
}

// withShow runs the load → authorise → decide → save recipe, retried on a
// conflict. promoterID "" skips the promoter check (system-driven commands).
func withShow(ctx context.Context, shows ShowRepository, useCase, showID, promoterID string, decide func(*domain.Show) error) error {
	id, err := domain.ParseShowID(showID)
	if err != nil {
		return fmt.Errorf("%s: %w", useCase, err)
	}
	err = RetryOnConflict(ctx, conflictAttempts, func(ctx context.Context) error {
		show, err := shows.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		if promoterID != "" {
			if err := authorise(show, promoterID); err != nil {
				return err
			}
		}
		if err := decide(show); err != nil {
			return err
		}
		if err := shows.Save(ctx, show); err != nil {
			return fmt.Errorf("save: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("%s: %w", useCase, err)
	}
	return nil
}

// authorise checks the caller drafted the show. It is an application rule:
// about who may run the use case, not about what a show is.
func authorise(show *domain.Show, promoterID string) error {
	promoter, err := domain.ParsePromoterID(promoterID)
	if err != nil {
		return err
	}
	if show.PromoterID() != promoter {
		return fmt.Errorf("%w: show %s", ErrNotPromoter, show.ID())
	}
	return nil
}

func newPriceList(specs []PriceSpec) (domain.PriceList, error) {
	prices := make(map[string]sharedkernel.Money, len(specs))
	for _, p := range specs {
		currency, err := sharedkernel.NewCurrency(p.Currency)
		if err != nil {
			return domain.PriceList{}, err
		}
		money, err := sharedkernel.NewMoney(p.Amount, currency)
		if err != nil {
			return domain.PriceList{}, err
		}
		prices[p.Section] = money
	}
	return domain.NewPriceList(prices)
}
