package memory

import (
	"context"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowQueries implements application.ShowQueries over a ShowRepository.
type ShowQueries struct{ repo *ShowRepository }

// NewShowQueries reads from repo.
func NewShowQueries(repo *ShowRepository) ShowQueries { return ShowQueries{repo: repo} }

// Get implements application.ShowQueries.
func (q ShowQueries) Get(ctx context.Context, id domain.ShowID) (application.ShowView, error) {
	s, err := q.repo.Get(ctx, id)
	if err != nil {
		return application.ShowView{}, err
	}
	return application.NewShowView(s), nil
}
