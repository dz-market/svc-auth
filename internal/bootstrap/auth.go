package bootstrap

import (
	"fmt"

	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/argon2id"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/jwt"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/opaque"
)

type tokenProviders struct {
	accessKey             jwt.KeyPair
	accessTokenIssuer     *jwt.Issuer
	accessTokenVerifier   *jwt.Verifier
	refreshTokenGenerator *opaque.Generator
}

func newTokenProviders(cfg config.Auth) (tokenProviders, error) {
	accessKey, err := jwt.ParseKeyPair(cfg.Access.PrivateKeyPEM)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("load access token key pair: %w", err)
	}

	accessTokenIssuer, err := jwt.NewIssuer(
		jwt.IssuerOptions{
			Key:      accessKey,
			Issuer:   cfg.Access.Issuer,
			Audience: cfg.Access.Audience,
		},
	)
	if err != nil {
		return tokenProviders{}, fmt.Errorf("access token issuer: %w", err)
	}

	accessTokenVerifier, err := jwt.NewVerifier(
		jwt.VerifierOptions{
			Key:      accessKey,
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
		accessKey:             accessKey,
		accessTokenIssuer:     accessTokenIssuer,
		accessTokenVerifier:   accessTokenVerifier,
		refreshTokenGenerator: refreshTokenGenerator,
	}, nil
}

func (a app) newAuthService(tokens tokenProviders) *auth.Service {
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
			PasswordHasher:        newPasswordHasher(a.cfg.Auth.Password.Argon2ID),
			AccessTokenIssuer:     tokens.accessTokenIssuer,
			RefreshTokenGenerator: tokens.refreshTokenGenerator,
			AccessTokenTTL:        a.cfg.Auth.Access.TTL,
			SessionTTL:            a.cfg.Auth.Session.TTL,
			Log:                   a.log,
		},
	)
}

func newPasswordHasher(cfg config.Argon2ID) *argon2id.Hasher {
	return argon2id.New(
		argon2id.Params{
			//nolint:gosec // bounded by the validation on the config field
			MemoryKiB:   uint32(cfg.Memory.KiB()),
			Iterations:  cfg.Iterations,
			Parallelism: cfg.Parallelism,
			SaltLength:  cfg.SaltLength,
			KeyLength:   cfg.KeyLength,
			MaxInFlight: cfg.MaxInFlight,
		},
	)
}
