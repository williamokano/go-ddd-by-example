-- ADR-013 supersedes ADR-005: one inventory per (show, section), so holds in
-- different sections never contend for the same version. Existing data moves
-- across: a section is the prefix of its seat refs ("ORCH/A/12" → ORCH),
-- sold out when all its seats are, positioned in its original layout order.
-- A hold that spanned two sections (allowed before) keeps its first seat's
-- section; holds last minutes, so the next sweep expires it.
-- +goose Up
CREATE TABLE ticketing.section_inventories (
    show_id    UUID        NOT NULL,
    section    TEXT        NOT NULL,
    position   INT         NOT NULL,    -- the section's place in the layout
    starts_at  TIMESTAMPTZ NOT NULL,
    closed     BOOLEAN     NOT NULL DEFAULT false,
    sold_out   BOOLEAN     NOT NULL DEFAULT false,
    version    INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (show_id, section)
);

INSERT INTO ticketing.section_inventories (show_id, section, position, starts_at, closed, sold_out, version)
SELECT i.show_id, s.section, s.position, i.starts_at, i.closed, s.sold_out, 1
FROM ticketing.inventories i
JOIN (
    SELECT show_id,
           split_part(seat_ref, '/', 1)                                       AS section,
           dense_rank() OVER (PARTITION BY show_id ORDER BY min(position)) - 1 AS position,
           bool_and(state = 'sold')                                           AS sold_out
    FROM ticketing.seats
    GROUP BY show_id, split_part(seat_ref, '/', 1)
) s ON s.show_id = i.show_id;

ALTER TABLE ticketing.seats ADD COLUMN section TEXT;
UPDATE ticketing.seats SET section = split_part(seat_ref, '/', 1);
ALTER TABLE ticketing.seats
    ALTER COLUMN section SET NOT NULL,
    DROP CONSTRAINT seats_show_id_fkey,
    ADD FOREIGN KEY (show_id, section) REFERENCES ticketing.section_inventories (show_id, section);

ALTER TABLE ticketing.holds ADD COLUMN section TEXT;
UPDATE ticketing.holds SET section = split_part(seats[1], '/', 1);
ALTER TABLE ticketing.holds
    ALTER COLUMN section SET NOT NULL,
    DROP CONSTRAINT holds_show_id_fkey,
    ADD FOREIGN KEY (show_id, section) REFERENCES ticketing.section_inventories (show_id, section);

DROP TABLE ticketing.inventories;

-- +goose Down
CREATE TABLE ticketing.inventories (
    show_id    UUID        PRIMARY KEY,
    starts_at  TIMESTAMPTZ NOT NULL,
    closed     BOOLEAN     NOT NULL DEFAULT false,
    sold_out   BOOLEAN     NOT NULL DEFAULT false,
    version    INT         NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO ticketing.inventories (show_id, starts_at, closed, sold_out, version)
SELECT show_id, min(starts_at), bool_or(closed), bool_and(sold_out), 1
FROM ticketing.section_inventories GROUP BY show_id;

ALTER TABLE ticketing.holds DROP CONSTRAINT holds_show_id_section_fkey, DROP COLUMN section,
    ADD FOREIGN KEY (show_id) REFERENCES ticketing.inventories (show_id);
ALTER TABLE ticketing.seats DROP CONSTRAINT seats_show_id_section_fkey, DROP COLUMN section,
    ADD FOREIGN KEY (show_id) REFERENCES ticketing.inventories (show_id);
DROP TABLE ticketing.section_inventories;
