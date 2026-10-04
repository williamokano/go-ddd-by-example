package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/seatquerytest"
)

func TestSeatQueries_Contract(t *testing.T) {
	seatquerytest.Run(t, func(*testing.T) (application.InventoryRepository, application.SeatQueries) {
		repo := memory.NewInventoryRepository()
		return repo, memory.NewSeatQueries(repo)
	})
}
