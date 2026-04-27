-- +goose Up
CREATE SEQUENCE IF NOT EXISTS lake.accounts_id_seq;
CREATE TABLE IF NOT EXISTS lake.accounts (
    id INTEGER NOT NULL DEFAULT nextval('lake.accounts_id_seq'),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    opening_balance INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS lake.accounts;
DROP SEQUENCE IF EXISTS lake.accounts_id_seq;
