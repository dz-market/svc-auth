-- +goose Up
CREATE TABLE outbox
(
    id uuid PRIMARY KEY,
    topic text NOT NULL,
    key text NOT NULL,
    payload bytea NOT NULL,
    created_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE outbox;
