package memory

import (
	"context"

	"github.com/williamokano/go-ddd-by-example/internal/venue/application"
	"github.com/williamokano/go-ddd-by-example/internal/venue/domain"
)

// VenueQueries implements application.VenueQueries over a VenueRepository's
// stored state. (Postgres implements it with plain SQL instead.)
type VenueQueries struct{ repo *VenueRepository }

// NewVenueQueries reads from repo.
func NewVenueQueries(repo *VenueRepository) VenueQueries { return VenueQueries{repo: repo} }

// Get implements application.VenueQueries.
func (q VenueQueries) Get(ctx context.Context, id domain.VenueID) (application.VenueView, error) {
	v, err := q.repo.Get(ctx, id)
	if err != nil {
		return application.VenueView{}, err
	}
	return toView(v), nil
}

func toView(v *domain.Venue) application.VenueView {
	view := application.VenueView{
		ID:       v.ID().String(),
		Name:     v.Name(),
		Street:   v.Address().Street(),
		City:     v.Address().City(),
		Country:  v.Address().Country(),
		Status:   v.Status().String(),
		Capacity: v.Capacity(),
	}
	for _, s := range v.Sections() {
		sv := application.SectionView{Code: s.Code().String(), Name: s.Name(), Kind: application.KindSeated, Capacity: s.Capacity()}
		if s.Kind() == domain.GeneralAdmission {
			sv.Kind = application.KindGA
		}
		for _, r := range s.Rows() {
			sv.Rows = append(sv.Rows, application.RowView{Label: r.Label(), Seats: r.Seats()})
		}
		view.Sections = append(view.Sections, sv)
	}
	return view
}
