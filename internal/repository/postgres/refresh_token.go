package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dz-market/svc-auth/internal/domain"
)

const refreshTokensTable = "refresh_tokens"

type RefreshTokenRepo struct {
	pool *pgxpool.Pool
	sb   squirrel.StatementBuilderType
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{
		pool: pool,
		sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (r *RefreshTokenRepo) Store(ctx context.Context, t domain.RefreshToken) error {
	query, args, err := r.sb.
		Insert(refreshTokensTable).
		Columns("id", "user_id", "session_id", "token_hash", "expires_at").
		Values(t.ID, t.UserID, t.SessionID, t.TokenHash, t.ExpiresAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build store refresh token query: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepo) FindByHash(ctx context.Context, hash []byte) (domain.RefreshToken, error) {
	query, args, err := r.sb.
		Select("id", "user_id", "session_id", "token_hash", "expires_at", "revoked_at", "created_at").
		From(refreshTokensTable).
		Where(squirrel.Eq{"token_hash": hash}).
		ToSql()
	if err != nil {
		return domain.RefreshToken{}, fmt.Errorf("build find refresh token by hash query: %w", err)
	}

	var t domain.RefreshToken

	err = r.pool.
		QueryRow(ctx, query, args...).
		Scan(&t.ID, &t.UserID, &t.SessionID, &t.TokenHash, &t.ExpiresAt, &t.RevokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
		}

		return domain.RefreshToken{}, fmt.Errorf("find refresh token by hash: %w", err)
	}

	return t, nil
}

func (r *RefreshTokenRepo) Rotate(ctx context.Context, oldHash []byte, newToken domain.RefreshToken) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin rotate tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op
	}()

	revokeSQL, revokeArgs, err := r.sb.
		Update(refreshTokensTable).
		Set("revoked_at", squirrel.Expr("now()")).
		Where(
			squirrel.Eq{
				"token_hash": oldHash,
				"revoked_at": nil,
			},
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke old token query: %w", err)
	}

	tag, err := tx.Exec(ctx, revokeSQL, revokeArgs...)
	if err != nil {
		return fmt.Errorf("revoke old token: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrRefreshTokenReused
	}

	insertSQL, insertArgs, err := r.sb.
		Insert(refreshTokensTable).
		Columns("id", "user_id", "session_id", "token_hash", "expires_at").
		Values(newToken.ID, newToken.UserID, newToken.SessionID, newToken.TokenHash, newToken.ExpiresAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build store new token query: %w", err)
	}

	if _, err := tx.Exec(ctx, insertSQL, insertArgs...); err != nil {
		return fmt.Errorf("store new token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit rotate: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepo) Revoke(ctx context.Context, hash []byte) error {
	query, args, err := r.sb.
		Update(refreshTokensTable).
		Set("revoked_at", squirrel.Expr("now()")).
		Where(
			squirrel.Eq{
				"token_hash": hash,
				"revoked_at": nil,
			},
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke token query: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("revoke token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepo) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query, args, err := r.sb.
		Update(refreshTokensTable).
		Set("revoked_at", squirrel.Expr("now()")).
		Where(
			squirrel.Eq{
				"session_id": sessionID,
				"revoked_at": nil,
			},
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke session query: %w", err)
	}

	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}
