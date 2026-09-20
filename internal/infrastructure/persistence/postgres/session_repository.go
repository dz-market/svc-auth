package postgres

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

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

func (r *SessionRepository) ByID(ctx context.Context, id uuid.UUID) (session.Session, error) {
	const query = `
		SELECT id, user_id, created_at, expires_at, revoked_at
		FROM sessions
		WHERE id = $1
	`

	var s session.Session

	if err := r.q.
		QueryRow(ctx, query, id).
		Scan(&s.ID, &s.UserID, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return session.Session{}, session.ErrSessionNotFound
		}

		return session.Session{}, fmt.Errorf("get session by id: %w", err)
	}

	return s, nil
}
