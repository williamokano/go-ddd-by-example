-- name: GetOrder :one
SELECT id, show_id, hold_id, customer_id, contact_email, lines, total_amount, currency, status, payment_ref, version, subtotal_amount, fee_amount, vat_amount
FROM ticketing.orders WHERE id = $1;

-- name: ListPaidOrdersForShow :many
SELECT id, show_id, hold_id, customer_id, contact_email, lines, total_amount, currency, status, payment_ref, version, subtotal_amount, fee_amount, vat_amount
FROM ticketing.orders WHERE show_id = $1 AND status IN ('paid', 'fulfilled');

-- name: InsertOrder :execrows
INSERT INTO ticketing.orders (id, show_id, hold_id, customer_id, contact_email, lines, total_amount, currency, status, payment_ref, version,
                              subtotal_amount, fee_amount, vat_amount)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 1, $11, $12, $13)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateOrder :execrows
UPDATE ticketing.orders SET status = $2, payment_ref = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND version = sqlc.arg(expected_version);

-- name: InsertTicket :exec
INSERT INTO ticketing.tickets (id, code, show_id, order_id, seat_ref, status, version)
VALUES ($1, $2, $3, $4, $5, $6, 1)
ON CONFLICT (id) DO NOTHING;

-- name: UpdateTicket :execrows
UPDATE ticketing.tickets SET status = $2, checked_in_at = $3, gate = $4, version = version + 1
WHERE id = $1 AND version = sqlc.arg(expected_version);

-- name: ListTicketsByOrder :many
SELECT id, code, show_id, order_id, seat_ref, status, version, checked_in_at, gate FROM ticketing.tickets WHERE order_id = $1 ORDER BY seat_ref;

-- name: ListTicketsByShow :many
SELECT id, code, show_id, order_id, seat_ref, status, version, checked_in_at, gate FROM ticketing.tickets WHERE show_id = $1 ORDER BY seat_ref;

-- name: GetTicketByCode :one
SELECT id, code, show_id, order_id, seat_ref, status, version, checked_in_at, gate FROM ticketing.tickets WHERE code = $1;
