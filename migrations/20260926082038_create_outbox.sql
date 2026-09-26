-- +goose Up
CREATE TABLE outbox
(
    id uuid PRIMARY KEY,
    topic text NOT NULL,
    key text NOT NULL,
    payload bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    attempts integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error text,
    published_at timestamptz,
    failed_at timestamptz
);

CREATE INDEX outbox_pending_idx ON outbox (created_at, id) WHERE published_at IS NULL AND failed_at IS NULL;

-- +goose Down
DROP TABLE outbox;
