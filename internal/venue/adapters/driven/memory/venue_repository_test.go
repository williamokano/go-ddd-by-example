package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/williamokano/go-ddd-by-example/internal/venue/adapters/driven/memory"
	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

func TestVenueRepository_Get_UnknownID(t *testing.T) {
	repo := memory.NewVenueRepository()

	_, err := repo.Get(context.Background(), domain.NewVenueID(uuid.New()))

	if !errors.Is(err, application.ErrVenueNotFound) {
		t.Errorf("Get() error = %v, want %v", err, application.ErrVenueNotFound)
	}
}
