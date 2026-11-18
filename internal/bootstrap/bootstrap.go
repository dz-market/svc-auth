package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/errgroup"

	phealth "github.com/dz-market/platform/grpc/health"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
)

type app struct {
	cfg                 config.Config
	log                 *slog.Logger
	postgres            *pgxpool.Pool
	kafkaProducerClient *kgo.Client
}

func Run(ctx context.Context, version string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)
	if err != nil {
		return err
	}

	log, err := logger.New(
		logger.Options{
			Format:  cfg.Log.Format,
			Level:   cfg.Log.Level,
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

	postgres, err := newPostgres(ctx, cfg.Postgres, cfg.ServiceName)
	if err != nil {
		return err
	}

	defer postgres.Close()

	kafkaProducerClient, err := newKafkaProducerClient(cfg, log)
	if err != nil {
		return err
	}

	defer kafkaProducerClient.Close()

	tokens, err := newTokenProviders(cfg.Auth)
	if err != nil {
		return err
	}

	log.InfoContext(
		ctx, "access key loaded",
		slog.String("kid", tokens.accessKey.ID),
	)

	a := app{
		cfg:                 cfg,
		log:                 log,
		postgres:            postgres,
		kafkaProducerClient: kafkaProducerClient,
	}

	authService, err := a.newAuthService(tokens)
	if err != nil {
		return err
	}

	relay, err := a.newRelay()
	if err != nil {
		return err
	}

	grpcServer, err := a.newGRPCServer(tokens, authService)
	if err != nil {
		return err
	}

	checker, err := phealth.New(
		grpcServer.Registrar(), phealth.Options{
			Period:  cfg.Health.Period,
			Timeout: cfg.Health.Timeout,
		},
	)
	if err != nil {
		return fmt.Errorf("create grpc health checker: %w", err)
	}

	if err := checker.Register("postgres", phealth.DependencyOptions{}, postgres.Ping); err != nil {
		return fmt.Errorf("register postgres health check: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return checker.Run(ctx)
		},
	)

	g.Go(
		func() error {
			return grpcServer.Run(
				ctx, func() {
					shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.Health.Timeout)
					defer cancel()

					if err := checker.Shutdown(shutdownCtx); err != nil {
						log.WarnContext(
							shutdownCtx, "grpc health shutdown failed",
							slog.Any("err", err),
						)
					}
				},
			)
		},
	)

	g.Go(
		func() error {
			kafka.EnsureTopics(
				ctx, kafkaProducerClient, []kafka.Topic{
					{
						Name:       kafka.TopicUserRegistered,
						Partitions: cfg.Kafka.Topics.UserRegistered.Partitions,
					},
				}, log,
			)

			return relay.Run(ctx)
		},
	)

	if err := g.Wait(); err != nil {
		return err
	}

	log.InfoContext(ctx, "service stopped")

	return nil
}
