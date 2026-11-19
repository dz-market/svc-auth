package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	trmpgx "github.com/avito-tech/go-transaction-manager/drivers/pgxv5/v2"
	"github.com/avito-tech/go-transaction-manager/trm/v2/manager"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
	"golang.org/x/sync/errgroup"

	phealth "github.com/dz-market/platform/grpc/health"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/server"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka/producer"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/argon2id"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

type app struct {
	grpcServer    *server.Server
	checker       *phealth.Checker
	relay         *outbox.Relay
	healthTimeout time.Duration
	log           *slog.Logger
}

func buildApp(
	ctx context.Context, cfg config.Config, postgresPool *pgxpool.Pool,
	kafkaProducerClient *kgo.Client, log *slog.Logger,
) (*app, error) {
	tokens, err := buildTokens(cfg.Auth)
	if err != nil {
		return nil, err
	}

	passwordHasher := argon2id.New(
		argon2id.Params{
			//nolint:gosec // bounded by the validation on the config field
			MemoryKiB:   uint32(cfg.Auth.Password.Argon2ID.Memory.KiB()),
			Iterations:  cfg.Auth.Password.Argon2ID.Iterations,
			Parallelism: cfg.Auth.Password.Argon2ID.Parallelism,
			SaltLength:  cfg.Auth.Password.Argon2ID.SaltLength,
			KeyLength:   cfg.Auth.Password.Argon2ID.KeyLength,
			MaxInFlight: cfg.Auth.Password.Argon2ID.MaxInFlight,
		},
	)

	log.InfoContext(
		ctx, "access key loaded",
		slog.String("kid", tokens.access.key.ID),
	)

	tx, err := manager.New(trmpgx.NewDefaultFactory(postgresPool))
	if err != nil {
		return nil, fmt.Errorf("create postgres transaction manager: %w", err)
	}

	q := postgres.NewTxQuerier(postgresPool, trmpgx.DefaultCtxGetter)
	outboxRepo := postgres.NewOutboxRepository(q)

	authService := auth.New(
		auth.Options{
			Tx:                    tx,
			Users:                 postgres.NewUserRepository(q),
			Sessions:              postgres.NewSessionRepository(q),
			RefreshTokens:         postgres.NewRefreshTokenRepository(q),
			Outbox:                outboxRepo,
			PasswordHasher:        passwordHasher,
			AccessTokenIssuer:     tokens.access.issuer,
			RefreshTokenGenerator: tokens.refresh,
			AccessTokenTTL:        cfg.Auth.Access.TTL,
			SessionTTL:            cfg.Auth.Session.TTL,
			Log:                   log,
		},
	)

	relay := outbox.New(
		outbox.Options{
			Tx:           tx,
			Repo:         outboxRepo,
			Publisher:    producer.New(kafkaProducerClient),
			PollInterval: cfg.Outbox.PollInterval,
			BatchSize:    cfg.Outbox.BatchSize,
			BatchTimeout: cfg.Outbox.BatchTimeout,
			MaxAttempts:  cfg.Outbox.MaxAttempts,
			Log:          log,
		},
	)

	grpcSrv, checker, err := buildGRPC(cfg, tokens.access, authService, postgresPool.Ping, log)
	if err != nil {
		return nil, err
	}

	return &app{
		grpcServer:    grpcSrv,
		checker:       checker,
		relay:         relay,
		healthTimeout: cfg.Health.Timeout,
		log:           log,
	}, nil
}

func (a *app) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return a.checker.Run(ctx)
		},
	)

	g.Go(
		func() error {
			return a.grpcServer.Run(
				ctx, func() {
					shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), a.healthTimeout)
					defer cancel()

					if err := a.checker.Shutdown(shutdownCtx); err != nil {
						a.log.WarnContext(
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
			return a.relay.Run(ctx)
		},
	)

	return g.Wait()
}
