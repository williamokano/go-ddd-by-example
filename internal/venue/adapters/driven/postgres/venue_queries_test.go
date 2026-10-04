//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application/venuequerytest"
)

func TestVenueQueries_Contract(t *testing.T) {
	venuequerytest.Run(t, func(t *testing.T) (application.VenueRepository, application.VenueQueries) {
		pool := pgtest.New(t)
		return postgres.NewVenueRepository(pool), postgres.NewVenueQueries(pool)
	})
}
