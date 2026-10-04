//go:build integration

package postgres_test

import (
	"os"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application"
	"github.com/williamokano/go-ddd-by-example/internal/ticketing/application/inventoryrepotest"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestInventoryRepository_Contract(t *testing.T) {
	inventoryrepotest.Run(t, func(t *testing.T) application.InventoryRepository {
		return postgres.NewInventoryRepository(pgtest.New(t))
	})
}
