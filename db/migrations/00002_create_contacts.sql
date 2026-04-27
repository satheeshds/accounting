-- +goose Up
CREATE SEQUENCE IF NOT EXISTS lake.contacts_id_seq;
CREATE TABLE IF NOT EXISTS lake.contacts (
    id INTEGER NOT NULL DEFAULT nextval('lake.contacts_id_seq'),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS lake.contacts;
DROP SEQUENCE IF EXISTS lake.contacts_id_seq;
