package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/dz-market/svc-auth/internal/domain/session"
)

type RefreshTokenRepository struct {
	q Querier
}

func NewRefreshTokenRepository(q Querier) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		q: q,
	}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, t session.RefreshToken) error {
	const query = `
		INSERT INTO refresh_tokens (id, session_id, token_hash, issued_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.q.Exec(ctx, query, t.ID, t.SessionID, t.Hash, t.IssuedAt); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) ByHash(ctx context.Context, hash []byte) (session.RefreshToken, error) {
	const query = `
		SELECT id, session_id, token_hash, issued_at, used_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var rt session.RefreshToken

	if err := r.q.
		QueryRow(ctx, query, hash).
		Scan(&rt.ID, &rt.SessionID, &rt.Hash, &rt.IssuedAt, &rt.UsedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return session.RefreshToken{}, session.ErrRefreshTokenNotFound
		}

		return session.RefreshToken{}, fmt.Errorf("get refresh token by hash: %w", err)
	}

	return rt, nil
}

func (r *RefreshTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) (bool, error) {
	const query = `
		UPDATE refresh_tokens
		SET used_at = $1
		WHERE id = $2 AND used_at IS NULL
	`

	tag, err := r.q.Exec(ctx, query, usedAt, id)
	if err != nil {
		return false, fmt.Errorf("mark refresh token used: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}
