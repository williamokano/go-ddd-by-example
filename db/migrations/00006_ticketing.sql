-- +goose Up
CREATE SCHEMA ticketing;

-- One row per published show: the ShowInventory aggregate's root (ADR-005).
CREATE TABLE ticketing.inventories (
    show_id    UUID        PRIMARY KEY,
    starts_at  TIMESTAMPTZ NOT NULL,
    closed     BOOLEAN     NOT NULL DEFAULT false,
    sold_out   BOOLEAN     NOT NULL DEFAULT false,
    version    INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ticketing.seats (
    show_id      UUID    NOT NULL REFERENCES ticketing.inventories (show_id),
    seat_ref     TEXT    NOT NULL,          -- "ORCH/A/12", "FLOOR/GA/0457"
    position     INT     NOT NULL,          -- layout order
    price_amount BIGINT  NOT NULL,          -- minor units
    currency     CHAR(3) NOT NULL,
    state        TEXT    NOT NULL,          -- available | held | sold
    hold_id      UUID,
    order_id     UUID,
    PRIMARY KEY (show_id, seat_ref)
);

CREATE TABLE ticketing.holds (
    hold_id     UUID        PRIMARY KEY,
    show_id     UUID        NOT NULL REFERENCES ticketing.inventories (show_id),
    customer_id UUID        NOT NULL,
    seats       TEXT[]      NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX holds_by_show ON ticketing.holds (show_id);
CREATE INDEX holds_by_expiry ON ticketing.holds (expires_at);

CREATE TABLE ticketing.outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_id     UUID        NOT NULL UNIQUE,
    topic        TEXT        NOT NULL,
    msg_key      TEXT        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);
CREATE INDEX ticketing_outbox_unpublished ON ticketing.outbox (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE ticketing.outbox;
DROP TABLE ticketing.holds;
DROP TABLE ticketing.seats;
DROP TABLE ticketing.inventories;
DROP SCHEMA ticketing;
