-- 9.8: Show keeps the venue's country, to forward it to Ticketing for VAT.
-- +goose Up
ALTER TABLE show.venue_layouts ADD COLUMN country TEXT NOT NULL DEFAULT 'PT';  -- every venue so far is in Portugal

-- +goose Down
ALTER TABLE show.venue_layouts DROP COLUMN country;
