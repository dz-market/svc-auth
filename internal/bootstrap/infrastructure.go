package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
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
