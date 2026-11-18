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
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	poolCfg.ConnConfig.RuntimeParams["application_name"] = serviceName

	pingCtx, cancel := context.WithTimeout(ctx, cfg.PingTimeout)
	defer cancel()

	db, err := postgres.New(pingCtx, poolCfg)
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
