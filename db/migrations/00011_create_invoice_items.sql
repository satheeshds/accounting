-- +goose Up
CREATE SEQUENCE IF NOT EXISTS lake.invoice_items_id_seq;
CREATE TABLE IF NOT EXISTS lake.invoice_items (
    id INTEGER NOT NULL DEFAULT nextval('lake.invoice_items_id_seq'),
    invoice_id INTEGER NOT NULL,
    description TEXT NOT NULL,
    quantity DOUBLE NOT NULL DEFAULT 1,
    unit TEXT,
    unit_price INTEGER NOT NULL DEFAULT 0,
    amount INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS lake.invoice_items;
DROP SEQUENCE IF EXISTS lake.invoice_items_id_seq;
