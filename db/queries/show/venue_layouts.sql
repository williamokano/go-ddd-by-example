-- name: GetVenueLayout :one
SELECT venue_id, name, active, sections, country FROM show.venue_layouts WHERE venue_id = $1;

-- name: UpsertVenueLayout :exec
INSERT INTO show.venue_layouts (venue_id, name, active, sections, country)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (venue_id) DO UPDATE
SET name = EXCLUDED.name, active = EXCLUDED.active, sections = EXCLUDED.sections, country = EXCLUDED.country, updated_at = now();
