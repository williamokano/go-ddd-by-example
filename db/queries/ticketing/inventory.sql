-- name: GetInventory :one
SELECT show_id, starts_at, closed, sold_out, version FROM ticketing.inventories WHERE show_id = $1;

-- name: ListSeats :many
SELECT seat_ref, price_amount, currency, state, hold_id, order_id
FROM ticketing.seats WHERE show_id = $1 ORDER BY position;

-- name: ListHolds :many
SELECT hold_id, customer_id, seats, expires_at FROM ticketing.holds WHERE show_id = $1;

-- name: GetShowIDByHold :one
SELECT show_id FROM ticketing.holds WHERE hold_id = $1;

-- name: InsertInventory :execrows
INSERT INTO ticketing.inventories (show_id, starts_at, closed, sold_out, version)
VALUES ($1, $2, $3, $4, 1)
ON CONFLICT (show_id) DO NOTHING;

-- name: UpdateInventory :execrows
UPDATE ticketing.inventories
SET closed = $2, sold_out = $3, version = version + 1, updated_at = now()
WHERE show_id = $1 AND version = sqlc.arg(expected_version);

-- name: InsertSeats :copyfrom
INSERT INTO ticketing.seats (show_id, seat_ref, position, price_amount, currency, state, hold_id, order_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: UpdateSeat :exec
UPDATE ticketing.seats SET state = $3, hold_id = $4, order_id = $5
WHERE show_id = $1 AND seat_ref = $2;

-- name: DeleteHolds :exec
DELETE FROM ticketing.holds WHERE show_id = $1;

-- name: InsertHold :exec
INSERT INTO ticketing.holds (hold_id, show_id, customer_id, seats, expires_at) VALUES ($1, $2, $3, $4, $5);

-- The read side: seats straight from the table.
-- name: InventoryExists :one
SELECT EXISTS (SELECT 1 FROM ticketing.inventories WHERE show_id = $1);

-- name: ShowsWithExpiredHolds :many
SELECT DISTINCT show_id FROM ticketing.holds WHERE expires_at <= $1;
