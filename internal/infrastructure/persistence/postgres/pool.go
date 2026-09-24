package postgres

import (
	"context"
	"log/slog"
	"time"

	ppostgres "github.com/dz-market/platform/database/postgres"
)

type Options struct {
	AppName           string
	DSN               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
	PingTimeout       time.Duration
}

func New(ctx context.Context, opts Options, log *slog.Logger) (*ppostgres.DB, error) {
	return ppostgres.New(
		ctx, ppostgres.Options{
			AppName:           opts.AppName,
			DSN:               opts.DSN,
			MaxConns:          opts.MaxConns,
			MinConns:          opts.MinConns,
			MaxConnLifetime:   opts.MaxConnLifetime,
			MaxConnIdleTime:   opts.MaxConnIdleTime,
			HealthCheckPeriod: opts.HealthCheckPeriod,
			ConnectTimeout:    opts.ConnectTimeout,
			PingTimeout:       opts.PingTimeout,
		}, log,
	)
}
