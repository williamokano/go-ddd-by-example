package httpapi

import "github.com/williamokano/go-ddd-by-example/internal/venue/application"

// The JSON shapes of the public API. They are separate from the domain (and
// from the application's views) so renaming a domain type never breaks a client.

type registerVenueRequest struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type createdResponse struct {
	ID string `json:"id"`
}

type addSectionRequest struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Rows     []rowDTO `json:"rows,omitempty"`
	Capacity int      `json:"capacity,omitempty"`
}

type rowDTO struct {
	Label           string `json:"label"`
	Seats           int    `json:"seats"`
	AccessibleSeats []int  `json:"accessibleSeats,omitempty"`
}

type venueResponse struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Address  addressDTO   `json:"address"`
	Status   string       `json:"status"`
	Capacity int          `json:"capacity"`
	Sections []sectionDTO `json:"sections"`
}

type addressDTO struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type sectionDTO struct {
	Code     string   `json:"code"`
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Capacity int      `json:"capacity"`
	Rows     []rowDTO `json:"rows,omitempty"`
}

func toVenueResponse(v application.VenueView) venueResponse {
	resp := venueResponse{
		ID: v.ID, Name: v.Name, Status: v.Status, Capacity: v.Capacity,
		Address:  addressDTO{Street: v.Street, City: v.City, Country: v.Country},
		Sections: make([]sectionDTO, 0, len(v.Sections)),
	}
	for _, s := range v.Sections {
		sd := sectionDTO{Code: s.Code, Name: s.Name, Kind: s.Kind, Capacity: s.Capacity}
		for _, r := range s.Rows {
			sd.Rows = append(sd.Rows, rowDTO{Label: r.Label, Seats: r.Seats})
		}
		resp.Sections = append(resp.Sections, sd)
	}
	return resp
}
