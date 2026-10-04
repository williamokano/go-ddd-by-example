-- name: GetSectionInventory :one
SELECT show_id, section, position, starts_at, closed, sold_out, version, country
FROM ticketing.section_inventories WHERE show_id = $1 AND section = $2;

-- name: ListSectionInventories :many
SELECT show_id, section, position, starts_at, closed, sold_out, version, country
FROM ticketing.section_inventories WHERE show_id = $1 ORDER BY position;

-- name: ListSeats :many
SELECT seat_ref, price_amount, currency, state, hold_id, order_id, accessible
FROM ticketing.seats WHERE show_id = $1 AND section = $2 ORDER BY position;

-- name: ListHolds :many
SELECT hold_id, customer_id, seats, expires_at FROM ticketing.holds WHERE show_id = $1 AND section = $2;

-- name: GetSectionByHold :one
SELECT show_id, section FROM ticketing.holds WHERE hold_id = $1;

-- name: InsertSectionInventory :execrows
INSERT INTO ticketing.section_inventories (show_id, section, position, starts_at, closed, sold_out, version, country)
VALUES ($1, $2, $3, $4, $5, $6, 1, $7)
ON CONFLICT (show_id, section) DO NOTHING;

-- name: UpdateSectionInventory :execrows
UPDATE ticketing.section_inventories
SET closed = $3, sold_out = $4, version = version + 1, updated_at = now()
WHERE show_id = $1 AND section = $2 AND version = sqlc.arg(expected_version);

-- name: InsertSeats :copyfrom
INSERT INTO ticketing.seats (show_id, section, seat_ref, position, price_amount, currency, state, hold_id, order_id, accessible)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: UpdateSeat :exec
UPDATE ticketing.seats SET state = $3, hold_id = $4, order_id = $5
WHERE show_id = $1 AND seat_ref = $2;

-- name: DeleteHolds :exec
DELETE FROM ticketing.holds WHERE show_id = $1 AND section = $2;

-- name: InsertHold :exec
INSERT INTO ticketing.holds (hold_id, show_id, section, customer_id, seats, expires_at) VALUES ($1, $2, $3, $4, $5, $6);

-- The read side: seats straight from the tables, in layout order.
-- name: InventoryExists :one
SELECT EXISTS (SELECT 1 FROM ticketing.section_inventories WHERE show_id = $1);

-- name: ListShowSeats :many
SELECT s.seat_ref, s.price_amount, s.currency, s.state, s.accessible
FROM ticketing.seats s
JOIN ticketing.section_inventories i ON i.show_id = s.show_id AND i.section = s.section
WHERE s.show_id = $1
ORDER BY i.position, s.position;

-- name: SectionsWithExpiredHolds :many
SELECT DISTINCT show_id, section FROM ticketing.holds WHERE expires_at <= $1;

-- name: ShowStartsAt :one
SELECT starts_at FROM ticketing.section_inventories WHERE show_id = $1 LIMIT 1;
