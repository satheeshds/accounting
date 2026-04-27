-- +goose Up
-- Junction table: many-to-many transaction <-> bill/invoice/payout/recurring_payment_occurrence.
-- Valid document_type values are enforced by the application layer.
CREATE SEQUENCE IF NOT EXISTS lake.transaction_documents_id_seq;
CREATE TABLE IF NOT EXISTS lake.transaction_documents (
    id INTEGER NOT NULL DEFAULT nextval('lake.transaction_documents_id_seq'),
    transaction_id INTEGER NOT NULL,
    document_type TEXT NOT NULL,
    document_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS lake.transaction_documents;
DROP SEQUENCE IF EXISTS lake.transaction_documents_id_seq;
