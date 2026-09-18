package postgres

import (
	"context"
	"fmt"

	"github.com/dz-market/svc-auth/internal/domain/session"
)

type SessionRepository struct {
	q Querier
}

func NewSessionRepository(q Querier) *SessionRepository {
	return &SessionRepository{
		q: q,
	}
}

func (r *SessionRepository) Create(ctx context.Context, s session.Session) error {
	const query = `
		INSERT INTO sessions (id, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.q.Exec(ctx, query, s.ID, s.UserID, s.CreatedAt, s.ExpiresAt); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}
