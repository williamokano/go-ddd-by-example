-- name: GetVenue :one
SELECT id, name, street, city, country, status, version
FROM venue.venues
WHERE id = $1;

-- name: ListSections :many
SELECT code, name, kind, capacity, rows
FROM venue.sections
WHERE venue_id = $1
ORDER BY position;

-- A second registration of the same id inserts nothing: 0 rows ⇒ conflict.
-- name: InsertVenue :execrows
INSERT INTO venue.venues (id, name, street, city, country, status, version)
VALUES ($1, $2, $3, $4, $5, $6, 1)
ON CONFLICT (id) DO NOTHING;

-- 0 rows ⇒ someone else saved first ⇒ ErrConcurrentModification (ADR-011).
-- name: UpdateVenue :execrows
UPDATE venue.venues
SET name = $2, street = $3, city = $4, country = $5, status = $6,
    version = version + 1, updated_at = now()
WHERE id = $1 AND version = sqlc.arg(expected_version);

-- name: DeleteSections :exec
DELETE FROM venue.sections WHERE venue_id = $1;

-- name: InsertSection :exec
INSERT INTO venue.sections (venue_id, code, position, name, kind, capacity, rows)
VALUES ($1, $2, $3, $4, $5, $6, $7);
