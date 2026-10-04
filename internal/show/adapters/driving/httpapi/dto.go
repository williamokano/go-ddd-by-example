package httpapi

import (
	"time"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
)

type draftShowRequest struct {
	PromoterID string    `json:"promoterId"`
	VenueID    string    `json:"venueId"`
	Title      string    `json:"title"`
	DoorsOpen  time.Time `json:"doorsOpen"`
	StartsAt   time.Time `json:"startsAt"`
	EndsAt     time.Time `json:"endsAt"`
}

type priceShowRequest struct {
	PromoterID string     `json:"promoterId"`
	Prices     []priceDTO `json:"prices"`
}

type priceDTO struct {
	Section  string `json:"section"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

type promoterRequest struct {
	PromoterID string `json:"promoterId"`
}

type cancelShowRequest struct {
	PromoterID string `json:"promoterId"`
	Reason     string `json:"reason"`
}

type createdResponse struct {
	ID string `json:"id"`
}

type showResponse struct {
	ID                 string     `json:"id"`
	VenueID            string     `json:"venueId"`
	PromoterID         string     `json:"promoterId"`
	Title              string     `json:"title"`
	DoorsOpen          time.Time  `json:"doorsOpen"`
	StartsAt           time.Time  `json:"startsAt"`
	EndsAt             time.Time  `json:"endsAt"`
	Status             string     `json:"status"`
	CancellationReason string     `json:"cancellationReason,omitempty"`
	Prices             []priceDTO `json:"prices"`
}

func toShowResponse(v application.ShowView) showResponse {
	resp := showResponse{
		ID: v.ID, VenueID: v.VenueID, PromoterID: v.PromoterID, Title: v.Title,
		DoorsOpen: v.DoorsOpen, StartsAt: v.StartsAt, EndsAt: v.EndsAt,
		Status: v.Status, CancellationReason: v.CancellationReason, Prices: make([]priceDTO, 0, len(v.Prices)),
	}
	for _, p := range v.Prices {
		resp.Prices = append(resp.Prices, priceDTO{Section: p.Section, Amount: p.Amount, Currency: p.Currency})
	}
	return resp
}
