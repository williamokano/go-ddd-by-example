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

type placeOrderRequest struct {
	HoldID       string `json:"holdId"`
	CustomerID   string `json:"customerId"`
	ContactEmail string `json:"contactEmail"`
}

type placeOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type orderResponse struct {
	ID      string           `json:"id"`
	ShowID  string           `json:"showId"`
	Status  string           `json:"status"`
	Total   priceDTO         `json:"total"`
	Tickets []ticketResponse `json:"tickets"`
}

type ticketResponse struct {
	Seat   string `json:"seat"`
	Code   string `json:"code"`
	Status string `json:"status"`
}
