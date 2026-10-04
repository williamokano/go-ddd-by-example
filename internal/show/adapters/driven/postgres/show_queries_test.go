//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/showquerytest"
)

func TestShowQueries_Contract(t *testing.T) {
	showquerytest.Run(t, func(t *testing.T) (application.ShowRepository, application.ShowQueries) {
		pool := pgtest.New(t)
		return postgres.NewShowRepository(pool), postgres.NewShowQueries(pool)
	})
}
