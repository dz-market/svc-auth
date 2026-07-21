package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dz-market/svc-auth/internal/domain"
)

const usersTable = "users"

type UserRepo struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *UserRepo) Create(ctx context.Context, user domain.User) error {
	query, args, err := r.sb.
		Insert(usersTable).
		Columns("id", "email", "password").
		Values(user.ID, user.Email.String(), user.Password).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create user query: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEmailTaken
		}

		return fmt.Errorf("create user: %w", err)
	}

	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email domain.Email) (domain.User, error) {
	query, args, err := r.sb.
		Select("id", "email", "password", "created_at", "updated_at").
		From(usersTable).
		Where(squirrel.Eq{"email": email.String()}).
		ToSql()
	if err != nil {
		return domain.User{}, fmt.Errorf("build find user by email query: %w", err)
	}

	var (
		user     domain.User
		rawEmail string
	)

	err = r.pool.
		QueryRow(ctx, query, args...).
		Scan(&user.ID, &rawEmail, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrUserNotFound
		}

		return domain.User{}, fmt.Errorf("find user by email: %w", err)
	}

	user.Email = domain.Email(rawEmail)

	return user, nil
}
