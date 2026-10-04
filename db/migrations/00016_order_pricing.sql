-- 9.8 (TKT-15): an order keeps its pricing breakdown. total_amount stays the
-- total; orders placed before fees and VAT were face value, so their
-- subtotal is their total.
-- +goose Up
ALTER TABLE ticketing.orders
    ADD COLUMN subtotal_amount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN fee_amount      BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN vat_amount      BIGINT NOT NULL DEFAULT 0;
UPDATE ticketing.orders SET subtotal_amount = total_amount;
ALTER TABLE ticketing.section_inventories ADD COLUMN country TEXT NOT NULL DEFAULT 'PT';  -- every venue so far is in Portugal

-- +goose Down
ALTER TABLE ticketing.section_inventories DROP COLUMN country;
ALTER TABLE ticketing.orders DROP COLUMN vat_amount, DROP COLUMN fee_amount, DROP COLUMN subtotal_amount;
