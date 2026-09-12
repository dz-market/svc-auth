package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultConnectTimeout = 5 * time.Second

type Options struct {
	DSN               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
	ConnectTimeout    time.Duration
	PingTimeout       time.Duration
}

type DB struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func New(ctx context.Context, opts Options, log *slog.Logger) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	applyOptions(cfg, opts)

	if opts.PingTimeout > 0 {
		cfg.PingTimeout = opts.PingTimeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	db := &DB{
		pool: pool,
		log:  log,
	}

	timeout := opts.ConnectTimeout
	if timeout <= 0 {
		timeout = defaultConnectTimeout
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping pool: %w", err)
	}

	log.InfoContext(
		ctx, "postgres connected",
		"host", cfg.ConnConfig.Host,
		"port", cfg.ConnConfig.Port,
		"database", cfg.ConnConfig.Database,
		"user", cfg.ConnConfig.User,
		"max_conns", cfg.MaxConns,
	)

	return db, nil
}

func (d *DB) Ping(ctx context.Context) error {
	if err := d.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}

func (d *DB) Close() {
	d.pool.Close()

	d.log.Info("postgres pool closed")
}

func applyOptions(cfg *pgxpool.Config, opts Options) {
	if opts.MaxConns > 0 {
		cfg.MaxConns = opts.MaxConns
	}

	if opts.MinConns > 0 {
		cfg.MinConns = opts.MinConns
	}

	if opts.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = opts.MaxConnLifetime
	}

	if opts.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = opts.MaxConnIdleTime
	}

	if opts.HealthCheckPeriod > 0 {
		cfg.HealthCheckPeriod = opts.HealthCheckPeriod
	}
}
