package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/dz-market/svc-auth/internal/domain/user"
)

type UserRepository struct {
	q Querier
}

func NewUserRepository(q Querier) *UserRepository {
	return &UserRepository{
		q: q,
	}
}

func (r *UserRepository) Create(ctx context.Context, u user.User) error {
	const query = `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.q.Exec(ctx, query, u.ID, u.Email, u.PasswordHash, u.CreatedAt); err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation {
			return fmt.Errorf("create user: %w", user.ErrEmailTaken)
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}
