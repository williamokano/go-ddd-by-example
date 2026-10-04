-- 9.4: the checkout as a process manager, one row per order.
-- +goose Up
CREATE TABLE ticketing.checkout_processes (
    order_id   UUID        PRIMARY KEY,
    show_id    UUID        NOT NULL,
    state      TEXT        NOT NULL,   -- started | awaiting_seats | completed | compensated
    version    INT         NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE ticketing.checkout_processes;
