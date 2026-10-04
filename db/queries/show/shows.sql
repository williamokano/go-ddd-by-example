-- name: GetShow :one
SELECT id, venue_id, promoter_id, title, doors_open, starts_at, ends_at, status, prices, cancellation_reason, version
FROM show.shows WHERE id = $1;

-- name: ListOpenShowsAtVenue :many
SELECT id, venue_id, promoter_id, title, doors_open, starts_at, ends_at, status, prices, cancellation_reason, version
FROM show.shows
WHERE venue_id = $1 AND status NOT IN ('cancelled', 'completed')
ORDER BY starts_at;

-- name: InsertShow :execrows
INSERT INTO show.shows (id, venue_id, promoter_id, title, doors_open, starts_at, ends_at, status, prices, cancellation_reason, version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateShow :execrows
UPDATE show.shows
SET title = $2, doors_open = $3, starts_at = $4, ends_at = $5, status = $6, prices = $7,
    cancellation_reason = $8, version = version + 1, updated_at = now()
WHERE id = $1 AND version = sqlc.arg(expected_version);

-- name: ListEndedShows :many
SELECT id FROM show.shows WHERE status IN ('published', 'sold_out') AND ends_at <= $1;
