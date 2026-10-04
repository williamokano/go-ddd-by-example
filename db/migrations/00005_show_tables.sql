-- +goose Up
CREATE TABLE show.shows (
    id                  UUID        PRIMARY KEY,
    venue_id            UUID        NOT NULL,    -- by id only: no FK across schemas
    promoter_id         UUID        NOT NULL,
    title               TEXT        NOT NULL,
    doors_open          TIMESTAMPTZ NOT NULL,
    starts_at           TIMESTAMPTZ NOT NULL,
    ends_at             TIMESTAMPTZ NOT NULL,
    status              TEXT        NOT NULL,
    prices              JSONB       NOT NULL DEFAULT '{}',  -- {"ORCH": {"amount": 4500, "currency": "EUR"}}
    cancellation_reason TEXT        NOT NULL DEFAULT '',
    version             INT         NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX shows_open_by_venue ON show.shows (venue_id) WHERE status NOT IN ('cancelled', 'completed');

CREATE TABLE show.outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_id     UUID        NOT NULL UNIQUE,
    topic        TEXT        NOT NULL,
    msg_key      TEXT        NOT NULL,
    event_type   TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ
);
CREATE INDEX show_outbox_unpublished ON show.outbox (id) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE show.outbox;
DROP TABLE show.shows;
