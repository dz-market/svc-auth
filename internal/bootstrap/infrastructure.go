package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
	"github.com/dz-market/svc-auth/internal/infrastructure/observability/health"
)

func newPostgres(ctx context.Context, cfg config.Config, log *slog.Logger) (*postgres.DB, error) {
	db, err := postgres.New(
		ctx, postgres.Options{
			AppName:           cfg.ServiceName,
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
		return nil, fmt.Errorf("postgres: %w", err)
	}

	return db, nil
}

func newKafkaProducerClient(cfg config.Config, log *slog.Logger) (*kgo.Client, error) {
	kafkaClient, err := kafka.NewProducerClient(
		kafka.ProducerOptions{
			Brokers:         cfg.Kafka.Brokers,
			ClientID:        cfg.ServiceName,
			DeliveryTimeout: cfg.Kafka.DeliveryTimeout,
			Log:             log,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("kafka: %w", err)
	}

	return kafkaClient, nil
}

func (a app) newChecker() *health.Checker {
	checker := health.New(
		health.Options{
			Period:  a.cfg.Health.Period,
			Timeout: a.cfg.Health.Timeout,
		}, a.log,
	)

	checker.Register("postgres", a.postgres.Ping)
	checker.Register("kafka", a.kafkaProducerClient.Ping)

	return checker
}
