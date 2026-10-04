//go:build integration

package postgres_test

import (
	"os"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application/venuerepotest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestVenueRepository_Contract(t *testing.T) {
	venuerepotest.Run(t, func(t *testing.T) application.VenueRepository {
		return postgres.NewVenueRepository(pgtest.New(t))
	})
}
