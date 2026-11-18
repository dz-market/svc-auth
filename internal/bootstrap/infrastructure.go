package bootstrap

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/config"
)

func newPostgres(ctx context.Context, cfg config.Postgres, serviceName string) (*pgxpool.Pool, error) {
	db, err := postgres.New(
		ctx, postgres.Options{
			DSN:                cfg.DSN,
			MaxConns:           &cfg.MaxConns,
			MinConns:           &cfg.MinConns,
			MaxConnLifetime:    &cfg.MaxConnLifetime,
			MaxConnIdleTime:    &cfg.MaxConnIdleTime,
			HealthCheckPeriod:  &cfg.HealthCheckPeriod,
			ConnectTimeout:     &cfg.ConnectTimeout,
			ApplicationName:    &serviceName,
			InitialPingTimeout: &cfg.PingTimeout,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	return db, nil
}
