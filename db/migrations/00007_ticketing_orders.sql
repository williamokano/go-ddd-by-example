-- +goose Up
CREATE TABLE ticketing.orders (
    id            UUID        PRIMARY KEY,
    show_id       UUID        NOT NULL,
    hold_id       UUID        NOT NULL,
    customer_id   UUID        NOT NULL,
    contact_email TEXT        NOT NULL,
    lines         JSONB       NOT NULL,       -- [{"seat": "ORCH/A/1", "amount": 4500, "currency": "EUR"}]
    total_amount  BIGINT      NOT NULL,
    currency      CHAR(3)     NOT NULL,
    status        TEXT        NOT NULL,
    payment_ref   TEXT        NOT NULL DEFAULT '',
    version       INT         NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX orders_by_show ON ticketing.orders (show_id);

CREATE TABLE ticketing.tickets (
    id       UUID PRIMARY KEY,                -- derived from order + seat (TKT-9)
    code     TEXT NOT NULL UNIQUE,
    show_id  UUID NOT NULL,
    order_id UUID NOT NULL,
    seat_ref TEXT NOT NULL,
    status   TEXT NOT NULL,                   -- valid | voided
    version  INT  NOT NULL
);
CREATE INDEX tickets_by_order ON ticketing.tickets (order_id);
CREATE INDEX tickets_by_show ON ticketing.tickets (show_id);

-- +goose Down
DROP TABLE ticketing.tickets;
DROP TABLE ticketing.orders;
