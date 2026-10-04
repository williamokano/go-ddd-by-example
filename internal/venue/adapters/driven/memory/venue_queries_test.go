package memory_test

import (
	"testing"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application/venuequerytest"
)

func TestVenueQueries_Contract(t *testing.T) {
	venuequerytest.Run(t, func(*testing.T) (application.VenueRepository, application.VenueQueries) {
		repo := memory.NewVenueRepository()
		return repo, memory.NewVenueQueries(repo)
	})
}
