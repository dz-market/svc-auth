package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	ppostgres "github.com/dz-market/platform/database/postgres"
	"github.com/dz-market/platform/logger"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
)

func Run(ctx context.Context, version string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)
	if err != nil {
		return err
	}

	log, err := logger.New(
		logger.Options{
			Format: logger.Format(cfg.Log.Format),
			HandlerOptions: slog.HandlerOptions{
				Level: cfg.Log.Level,
			},
			Service: cfg.ServiceName,
			Version: version,
		},
	)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	log.DebugContext(ctx, "configuration loaded")
	log.InfoContext(
		ctx, "service starting",
		slog.String("go_version", runtime.Version()),
		slog.Int("pid", os.Getpid()),
		slog.String("grpc_addr", cfg.GRPC.Addr),
		slog.String("log_level", cfg.Log.Level.String()),
	)

	postgresPool, err := ppostgres.New(
		ctx, ppostgres.Options{
			DSN:                cfg.Postgres.DSN,
			MaxConns:           &cfg.Postgres.MaxConns,
			MinConns:           &cfg.Postgres.MinConns,
			MaxConnLifetime:    &cfg.Postgres.MaxConnLifetime,
			MaxConnIdleTime:    &cfg.Postgres.MaxConnIdleTime,
			HealthCheckPeriod:  &cfg.Postgres.HealthCheckPeriod,
			ConnectTimeout:     &cfg.Postgres.ConnectTimeout,
			ApplicationName:    &cfg.ServiceName,
			InitialPingTimeout: &cfg.Postgres.PingTimeout,
		},
	)
	if err != nil {
		return fmt.Errorf("open postgres pool: %w", err)
	}
	defer postgresPool.Close()

	kafkaProducerClient, err := kafka.NewProducerClient(
		kafka.ProducerOptions{
			Brokers:         cfg.Kafka.Brokers,
			ClientID:        cfg.ServiceName,
			DeliveryTimeout: cfg.Kafka.DeliveryTimeout,
			Log:             log,
		},
	)
	if err != nil {
		return fmt.Errorf("create kafka producer client: %w", err)
	}
	defer kafkaProducerClient.Close()

	application, err := buildApp(ctx, cfg, postgresPool, kafkaProducerClient, log)
	if err != nil {
		return err
	}

	if err := application.run(ctx); err != nil {
		return err
	}

	log.InfoContext(ctx, "service stopped")

	return nil
}
