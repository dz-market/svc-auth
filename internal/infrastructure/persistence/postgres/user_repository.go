package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/domain/user"
)

type UserRepository struct {
	q ppostgres.Querier
}

func NewUserRepository(q ppostgres.Querier) *UserRepository {
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
		if ppostgres.IsUniqueViolation(err, "") {
			return fmt.Errorf("create user: %w", user.ErrEmailTaken)
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepository) ByEmail(ctx context.Context, email string) (user.User, error) {
	const query = `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var u user.User

	if err := r.q.
		QueryRow(ctx, query, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user.User{}, user.ErrNotFound
		}

		return user.User{}, fmt.Errorf("get user by email: %w", err)
	}

	return u, nil
}
