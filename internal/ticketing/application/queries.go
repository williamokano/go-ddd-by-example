package application

import (
	"context"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/domain"
)

// SeatQueries is the read side of the inventory: what's on sale, straight
// from storage, with no aggregate load.
type SeatQueries interface {
	// ListSeats returns every seat of a show in layout order, or
	// ErrInventoryNotFound.
	ListSeats(ctx context.Context, showID domain.ShowID) ([]SeatRow, error)
}

// SeatRow is one seat as buyers see it.
type SeatRow struct {
	Ref      string
	State    string
	Amount   int64
	Currency string
}
