//go:build integration

package postgres_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/postgres"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/showrepotest"
)

func TestShowRepository_Contract(t *testing.T) {
	showrepotest.Run(t, func(t *testing.T) application.ShowRepository { return postgres.NewShowRepository(pgtest.New(t)) })
}
