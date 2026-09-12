package bootstrap

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/server"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
)

const serviceName = "svc-auth"

func Run(ctx context.Context, version string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(
		logger.Options{
			Level:   cfg.Log.Level,
			Format:  logger.Format(cfg.Log.Format),
			Service: serviceName,
			Version: version,
		},
	)

	db, err := postgres.New(
		ctx, postgres.Options{
			DSN:               cfg.Postgres.DSN,
			MaxConns:          cfg.Postgres.MaxConns,
			MinConns:          cfg.Postgres.MinConns,
			MaxConnLifetime:   cfg.Postgres.MaxConnLifetime,
			MaxConnIdleTime:   cfg.Postgres.MaxConnIdleTime,
			HealthCheckPeriod: cfg.Postgres.HealthCheckPeriod,
			ConnectTimeout:    cfg.Postgres.ConnectTimeout,
			PingTimeout:       cfg.Postgres.PingTimeout,
		}, log,
	)
	if err != nil {
		return err
	}

	defer db.Close()

	srv := server.New(
		server.Options{
			Addr:       cfg.GRPC.Addr,
			Reflection: cfg.GRPC.Reflection,
		},
		log,
	)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Serve(ctx)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("grpc server: %w", err)

	case <-ctx.Done():
		log.InfoContext(ctx, "shutdown requested")
	}

	srv.Shutdown(ctx, cfg.ShutdownTimeout)

	return nil
}
