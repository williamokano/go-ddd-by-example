-- name: GetCheckoutProcess :one
SELECT order_id, show_id, state, version FROM ticketing.checkout_processes WHERE order_id = $1;

-- name: InsertCheckoutProcess :execrows
INSERT INTO ticketing.checkout_processes (order_id, show_id, state, version) VALUES ($1, $2, $3, 1)
ON CONFLICT (order_id) DO NOTHING;

-- name: UpdateCheckoutProcess :execrows
UPDATE ticketing.checkout_processes SET state = $2, version = version + 1, updated_at = now()
WHERE order_id = $1 AND version = sqlc.arg(expected_version);
