package postgres

import (
	"context"
	"fmt"

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
