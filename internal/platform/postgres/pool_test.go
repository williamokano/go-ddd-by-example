//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/platform/postgres/pgtest"
)

func TestNew_GivesAWorkingPool(t *testing.T) {
	pool := pgtest.New(t)

	var one int
	if err := pool.QueryRow(context.Background(), "SELECT 1").Scan(&one); err != nil || one != 1 {
		t.Errorf("SELECT 1 = %d, %v; want 1, nil", one, err)
	}
}
