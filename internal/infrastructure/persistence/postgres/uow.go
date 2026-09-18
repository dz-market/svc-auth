package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollbackTimeout = 5 * time.Second

type UnitOfWork[R any] struct {
	pool  *pgxpool.Pool
	repos func(Querier) R
}

func NewUnitOfWork[R any](db *DB, repos func(Querier) R) *UnitOfWork[R] {
	return &UnitOfWork[R]{
		pool:  db.pool,
		repos: repos,
	}
}

func (u *UnitOfWork[R]) Do(ctx context.Context, fn func(r R) error) (err error) {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
		defer cancel()

		if rollbackErr := tx.Rollback(rollbackCtx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("rollback: %w", rollbackErr))
		}
	}()

	if fnErr := fn(u.repos(tx)); fnErr != nil {
		return fnErr
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
