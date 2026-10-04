-- name: GetVenueLayout :one
SELECT venue_id, name, active, sections FROM show.venue_layouts WHERE venue_id = $1;

-- name: UpsertVenueLayout :exec
INSERT INTO show.venue_layouts (venue_id, name, active, sections)
VALUES ($1, $2, $3, $4)
ON CONFLICT (venue_id) DO UPDATE
SET name = EXCLUDED.name, active = EXCLUDED.active, sections = EXCLUDED.sections, updated_at = now();
