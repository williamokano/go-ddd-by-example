package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/williamokano/go-ddd-by-example/internal/show/application"
	"github.com/williamokano/go-ddd-by-example/internal/show/domain"
)

// ShowQueries implements application.ShowQueries. A show's row is already
// flat, so the read side reuses the GetShow query and renders the view.
type ShowQueries struct{ repo *ShowRepository }

// NewShowQueries returns the queries using pool.
func NewShowQueries(pool *pgxpool.Pool) *ShowQueries {
	return &ShowQueries{repo: NewShowRepository(pool)}
}

// Get implements application.ShowQueries.
func (q *ShowQueries) Get(ctx context.Context, id domain.ShowID) (application.ShowView, error) {
	s, err := q.repo.Get(ctx, id)
	if err != nil {
		return application.ShowView{}, err
	}
	return application.NewShowView(s), nil
}
