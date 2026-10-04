package postgres

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// showRow unifies the identical row types sqlc generates per query.
type showRow struct {
	ID, VenueID, PromoterID     uuid.UUID
	Title                       string
	DoorsOpen, StartsAt, EndsAt time.Time
	Status                      string
	Prices                      []byte
	CancellationReason          string
	Version                     int32
}

type priceJSON struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

var statuses = map[string]domain.Status{}

func init() {
	for _, s := range []domain.Status{domain.Draft, domain.Published, domain.SoldOut, domain.Cancelled, domain.Completed} {
		statuses[s.String()] = s
	}
}

func rowToShow(r showRow) (*domain.Show, error) {
	schedule, err := domain.NewSchedule(r.DoorsOpen, r.StartsAt, r.EndsAt)
	if err != nil {
		return nil, fmt.Errorf("show %s: stored schedule: %w", r.ID, err)
	}
	status, ok := statuses[r.Status]
	if !ok {
		return nil, fmt.Errorf("show %s: unknown stored status %q", r.ID, r.Status)
	}
	prices, err := pricesFrom(r.Prices)
	if err != nil {
		return nil, fmt.Errorf("show %s: %w", r.ID, err)
	}
	state := domain.ShowState{
		ID: domain.NewShowID(r.ID), VenueID: domain.NewVenueID(r.VenueID), PromoterID: domain.NewPromoterID(r.PromoterID),
		Title: r.Title, Schedule: schedule, Prices: prices, Status: status, Version: int(r.Version),
	}
	if r.CancellationReason != "" {
		if state.CancellationReason, err = domain.NewCancellationReason(r.CancellationReason); err != nil {
			return nil, fmt.Errorf("show %s: %w", r.ID, err)
		}
	}
	return domain.RehydrateShow(state), nil
}

func pricesFrom(b []byte) (domain.PriceList, error) {
	var stored map[string]priceJSON
	if err := json.Unmarshal(b, &stored); err != nil {
		return domain.PriceList{}, fmt.Errorf("stored prices: %w", err)
	}
	if len(stored) == 0 {
		return domain.PriceList{}, nil
	}
	prices := make(map[string]domain.Money, len(stored))
	for code, p := range stored {
		currency, err := domain.NewCurrency(p.Currency)
		if err != nil {
			return domain.PriceList{}, err
		}
		if prices[code], err = domain.NewMoney(p.Amount, currency); err != nil {
			return domain.PriceList{}, err
		}
	}
	return domain.NewPriceList(prices)
}

func pricesJSON(prices domain.PriceList) ([]byte, error) {
	stored := make(map[string]priceJSON)
	for _, code := range prices.Sections() {
		m, _ := prices.Price(code)
		stored[code] = priceJSON{Amount: m.Amount(), Currency: m.Currency().String()}
	}
	b, err := json.Marshal(stored)
	if err != nil {
		return nil, fmt.Errorf("prices: %w", err)
	}
	return b, nil
}
