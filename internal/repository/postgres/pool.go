package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dz-market/svc-auth/internal/config"
)

func NewPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	poolCfg.MinConns = cfg.Postgres.MinConns
	poolCfg.MaxConns = cfg.Postgres.MaxConns
	poolCfg.MaxConnLifetime = cfg.Postgres.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.Postgres.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.Postgres.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return pool, nil
}
