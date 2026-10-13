-- +goose Up
CREATE TABLE trip_idempotency_keys (
    idempotency_key UUID PRIMARY KEY,
    trip_id         UUID NOT NULL UNIQUE REFERENCES trips(id)
        ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED,
    request_hash    BYTEA NOT NULL,
        CHECK (octet_length(request_hash) = 32),
    expires_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX idempotency_keys_expires_at_idx ON trip_idempotency_keys (expires_at);

-- +goose Down
DROP TABLE trip_idempotency_keys;
