package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/errgroup"

	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
)

type app struct {
	cfg                 config.Config
	log                 *slog.Logger
	postgres            *ppostgres.DB
	kafkaProducerClient *kgo.Client
}

func Run(ctx context.Context, version string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(ctx)
	if err != nil {
		return err
	}

	log := logger.New(
		logger.Options{
			Level:   cfg.Log.Level,
			Format:  cfg.Log.Format,
			Service: cfg.ServiceName,
			Version: version,
		},
	)

	log.DebugContext(ctx, "configuration loaded")

	log.InfoContext(
		ctx, "service starting",
		slog.String("go_version", runtime.Version()),
		slog.Int("pid", os.Getpid()),
		slog.String("grpc_addr", cfg.GRPC.Addr),
		slog.String("log_level", cfg.Log.Level.String()),
	)

	postgres, err := newPostgres(ctx, cfg, log)
	if err != nil {
		return err
	}

	defer postgres.Close()

	kafkaProducerClient, err := newKafkaProducerClient(cfg, log)
	if err != nil {
		return err
	}

	defer kafkaProducerClient.Close()

	tokenProviders, err := newTokenProviders(cfg.Auth)
	if err != nil {
		return err
	}

	log.InfoContext(
		ctx, "access key loaded",
		slog.String("kid", tokenProviders.key.ID),
	)

	a := app{
		cfg:                 cfg,
		log:                 log,
		postgres:            postgres,
		kafkaProducerClient: kafkaProducerClient,
	}

	authService := a.newAuthService(tokenProviders)
	relay := a.newRelay()
	checker := a.newChecker()

	grpcServer, err := a.newGRPCServer(tokenProviders, authService)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return checker.Run(
				ctx, func(healthy bool) {
					log.WarnContext(
						ctx, "healthy state changed",
						slog.Bool("healthy", healthy),
					)

					grpcServer.SetHealthy(healthy)
				},
			)
		},
	)

	g.Go(
		func() error {
			return grpcServer.Run(ctx)
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
