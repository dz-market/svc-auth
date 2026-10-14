package bootstrap

import (
	"fmt"

	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka/producer"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/argon2id"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/jwt"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/opaque"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

type tokenProviders struct {
	key                   jwt.KeyPair
	accessTokenIssuer     *jwt.Issuer
	accessTokenVerifier   *jwt.Verifier
	refreshTokenGenerator *opaque.Generator
}

func newTokenProviders(cfg config.Auth) (tokenProviders, error) {
	key, err := jwt.ParseKeyPair(cfg.Access.PrivateKey)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("load access token key pair: %w", err)
	}

	accessTokenIssuer, err := jwt.NewIssuer(
		jwt.IssuerOptions{
			Key:      key,
			Issuer:   cfg.Access.Issuer,
			Audience: cfg.Access.Audience,
		},
	)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("access token issuer: %w", err)
	}

	accessTokenVerifier, err := jwt.NewVerifier(
		jwt.VerifierOptions{
			Key:      key,
			Issuer:   cfg.Access.Issuer,
			Audience: cfg.Access.Audience,
		},
	)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("access token verifier: %w", err)
	}

	refreshTokenGenerator, err := opaque.New(
		opaque.Options{
			Length: cfg.Refresh.Length,
		},
	)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("refresh token generator: %w", err)
	}

	return tokenProviders{
		key:                   key,
		accessTokenIssuer:     accessTokenIssuer,
		accessTokenVerifier:   accessTokenVerifier,
		refreshTokenGenerator: refreshTokenGenerator,
	}, nil
}

func (a app) newAuthService(tokenProviders tokenProviders) *auth.Service {
	newAuthRepos := func(q ppostgres.Querier) auth.Repositories {
		return auth.Repositories{
			Users:         postgres.NewUserRepository(q),
			RefreshTokens: postgres.NewRefreshTokenRepository(q),
			Sessions:      postgres.NewSessionRepository(q),
			Outbox:        postgres.NewOutboxRepository(q),
		}
	}

	return auth.New(
		auth.Options{
			Repos:                 newAuthRepos(a.postgres),
			UoW:                   ppostgres.NewUnitOfWork(a.postgres, newAuthRepos),
			PasswordHasher:        newPasswordHasher(a.cfg.Auth),
			AccessTokenIssuer:     tokenProviders.accessTokenIssuer,
			RefreshTokenGenerator: tokenProviders.refreshTokenGenerator,
			AccessTokenTTL:        a.cfg.Auth.Access.TTL,
			SessionTTL:            a.cfg.Auth.Session.TTL,
			Log:                   a.log,
		},
	)
}

func newPasswordHasher(cfg config.Auth) *argon2id.Hasher {
	return argon2id.New(
		argon2id.Params{
			//nolint:gosec // bounded by the validation on the config field
			MemoryKiB:   uint32(cfg.Password.Argon2ID.Memory.KiB()),
			Iterations:  cfg.Password.Argon2ID.Iterations,
			Parallelism: cfg.Password.Argon2ID.Parallelism,
			SaltLength:  cfg.Password.Argon2ID.SaltLength,
			KeyLength:   cfg.Password.Argon2ID.KeyLength,
			MaxInFlight: cfg.Password.Argon2ID.MaxInFlight,
		},
	)
}

func (a app) newRelay() *outbox.Relay {
	return outbox.New(
		outbox.Options{
			UoW: ppostgres.NewUnitOfWork(
				a.postgres, func(q ppostgres.Querier) outbox.Store {
					return postgres.NewOutboxRepository(q)
				},
			),
			Publisher:    producer.New(a.kafkaProducerClient),
			PollInterval: a.cfg.Outbox.PollInterval,
			BatchSize:    a.cfg.Outbox.BatchSize,
			BatchTimeout: a.cfg.Outbox.BatchTimeout,
			MaxAttempts:  a.cfg.Outbox.MaxAttempts,
			Log:          a.log,
		},
	)
}
