package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"buf.build/go/protovalidate"
	"golang.org/x/sync/errgroup"

	ppostgres "github.com/dz-market/platform/database/postgres"
	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/handler"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/server"
	"github.com/dz-market/svc-auth/internal/infrastructure/clock"
	"github.com/dz-market/svc-auth/internal/infrastructure/observability/health"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/argon2id"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/jwt"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/opaque"
)

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
		return fmt.Errorf("postgres: %w", err)
	}

	defer db.Close()

	key, err := jwt.LoadKeyPair(cfg.Auth.Access.PrivateKeyPath, cfg.Auth.Access.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("load access key: %w", err)
	}

	log.InfoContext(
		ctx, "access key loaded",
		slog.String("kid", key.ID),
	)

	accessTokenIssuer, err := jwt.NewIssuer(
		jwt.IssuerOptions{
			Key:      key,
			Issuer:   cfg.Auth.Access.Issuer,
			Audience: cfg.Auth.Access.Audience,
		},
	)
	if err != nil {
		return fmt.Errorf("access token issuer: %w", err)
	}

	accessTokenVerifier, err := jwt.NewVerifier(
		jwt.VerifierOptions{
			Key:      key,
			Issuer:   cfg.Auth.Access.Issuer,
			Audience: cfg.Auth.Access.Audience,
		},
	)
	if err != nil {
		return fmt.Errorf("access token verifier: %w", err)
	}

	refreshTokenGenerator, err := opaque.New(
		opaque.Options{
			Length: cfg.Auth.Refresh.Length,
		},
	)
	if err != nil {
		return fmt.Errorf("refresh token generator: %w", err)
	}

	hasher := argon2id.New(
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

	systemClock := clock.System{}

	newAuthRepos := func(q ppostgres.Querier) auth.Repositories {
		return auth.Repositories{
			Users:         postgres.NewUserRepository(q),
			RefreshTokens: postgres.NewRefreshTokenRepository(q),
			Sessions:      postgres.NewSessionRepository(q),
			Outbox:        postgres.NewOutboxRepository(q),
		}
	}

	authService := auth.New(
		auth.Options{
			Repos:             newAuthRepos(db),
			UoW:               ppostgres.NewUnitOfWork(db, newAuthRepos),
			Hasher:            hasher,
			AccessTokenIssuer: accessTokenIssuer,
			RefreshGenerator:  refreshTokenGenerator,
			Clock:             systemClock,
			AccessTokenTTL:    cfg.Auth.Access.TTL,
			SessionTTL:        cfg.Auth.Session.TTL,
			Log:               log,
		},
	)

	checker := health.New(
		health.Options{
			Period:  cfg.Health.Period,
			Timeout: cfg.Health.Timeout,
		}, log,
	)
	checker.Register("postgres", db.Ping)

	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("protovalidate: %w", err)
	}

	srv := server.New(
		server.Options{
			Addr:                  cfg.GRPC.Addr,
			Reflection:            cfg.GRPC.Reflection,
			MaxRecvMsgSize:        cfg.GRPC.MaxRecvMsgSize.Bytes(),
			MaxConnectionAge:      cfg.GRPC.Keepalive.MaxConnectionAge,
			MaxConnectionAgeGrace: cfg.GRPC.Keepalive.MaxConnectionAgeGrace,
			Validator:             validator,
			Verifier:              accessTokenVerifier,
		},
		log,
	)

	authv1.RegisterAuthServiceServer(
		srv.Registrar(), handler.NewAuth(
			handler.Options{
				Service: authService,
				Clock:   systemClock,
				Log:     log,
			},
		),
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return checker.Run(
				ctx, func(healthy bool) {
					log.WarnContext(
						ctx, "healthy state changed",
						slog.Bool("healthy", healthy),
					)

					srv.SetHealthy(healthy)
				},
			)
		},
	)

	g.Go(
		func() error {
			return srv.Run(ctx)
		},
	)

	if err := g.Wait(); err != nil {
		return err
	}

	log.InfoContext(ctx, "service stopped")

	return nil
}
