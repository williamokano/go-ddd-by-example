-- The read side: flat views straight from SQL, never through the aggregate.

-- name: GetVenueView :one
SELECT v.id, v.name, v.street, v.city, v.country, v.status,
       COALESCE(SUM(s.capacity), 0)::int AS capacity
FROM venue.venues v
LEFT JOIN venue.sections s ON s.venue_id = v.id
WHERE v.id = $1
GROUP BY v.id;

-- name: ListVenueViewsByStatus :many
SELECT v.id, v.name, v.street, v.city, v.country, v.status,
       COALESCE(SUM(s.capacity), 0)::int AS capacity
FROM venue.venues v
LEFT JOIN venue.sections s ON s.venue_id = v.id
WHERE v.status = $1
GROUP BY v.id
ORDER BY v.name, v.id;

-- name: ListSectionsOfVenues :many
SELECT venue_id, code, name, kind, capacity, rows
FROM venue.sections
WHERE venue_id = ANY(sqlc.arg(venue_ids)::uuid[])
ORDER BY venue_id, position;
