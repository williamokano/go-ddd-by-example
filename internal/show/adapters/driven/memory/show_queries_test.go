package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/show/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/application/showquerytest"
)

func TestShowQueries_Contract(t *testing.T) {
	showquerytest.Run(t, func(*testing.T) (application.ShowRepository, application.ShowQueries) {
		repo := memory.NewShowRepository()
		return repo, memory.NewShowQueries(repo)
	})
}
