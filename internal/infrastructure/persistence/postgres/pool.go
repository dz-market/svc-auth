package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

	pingCtx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
	defer cancel()

	if err := db.Ping(pingCtx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping pool: %w", err)
	}

	log.InfoContext(
		ctx, "postgres connected",
		slog.String("host", cfg.ConnConfig.Host),
		slog.Int("port", int(cfg.ConnConfig.Port)),
		slog.String("database", cfg.ConnConfig.Database),
		slog.String("user", cfg.ConnConfig.User),
		slog.Int("max_conns", int(cfg.MaxConns)),
	)

	return db, nil
}

func (d *DB) Ping(ctx context.Context) error {
	if err := d.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	return nil
}

func (d *DB) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return d.pool.Exec(ctx, query, args...)
}

func (d *DB) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	return d.pool.Query(ctx, query, args...)
}

func (d *DB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return d.pool.QueryRow(ctx, query, args...)
}

func (d *DB) Close() {
	d.pool.Close()

	d.log.Info("postgres pool closed")
}

func applyOptions(cfg *pgxpool.Config, opts Options) {
	cfg.ConnConfig.RuntimeParams["application_name"] = opts.AppName

	if opts.ConnectTimeout > 0 {
		cfg.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	}

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
