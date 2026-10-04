package httpapi

import "time"

type holdRequest struct {
	CustomerID string   `json:"customerId"`
	Seats      []string `json:"seats"`
}

type holdResponse struct {
	HoldID    string    `json:"holdId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type customerRequest struct {
	CustomerID string `json:"customerId"`
}

type seatResponse struct {
	Ref   string   `json:"ref"`
	State string   `json:"state"`
	Price priceDTO `json:"price"`
}

type priceDTO struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}
