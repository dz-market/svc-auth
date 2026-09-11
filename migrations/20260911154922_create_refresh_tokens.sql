-- +goose Up
CREATE TABLE refresh_tokens
(
    id uuid PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,

    token_hash bytea NOT NULL UNIQUE,

    issued_at timestamptz NOT NULL DEFAULT now(),
    used_at timestamptz
);

CREATE INDEX refresh_tokens_session_id_idx ON refresh_tokens (session_id);

-- +goose Down
DROP TABLE refresh_tokens;