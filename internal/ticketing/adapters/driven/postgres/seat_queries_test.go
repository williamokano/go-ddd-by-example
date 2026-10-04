//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/seatquerytest"
)

func TestSeatQueries_Contract(t *testing.T) {
	seatquerytest.Run(t, func(t *testing.T) (application.InventoryRepository, application.SeatQueries) {
		pool := pgtest.New(t)
		return postgres.NewInventoryRepository(pool), postgres.NewSeatQueries(pool)
	})
}
