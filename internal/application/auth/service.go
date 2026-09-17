package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type Options struct {
	UoW               UnitOfWork
	Hasher            PasswordHasher
	AccessTokenIssuer AccessTokenIssuer
	RefreshGenerator  RefreshTokenGenerator
	Clock             Clock
	AccessTokenTTL    time.Duration
	SessionTTL        time.Duration
	Log               *slog.Logger
}

type Service struct {
	uow                   UnitOfWork
	hasher                PasswordHasher
	accessTokenIssuer     AccessTokenIssuer
	refreshTokenGenerator RefreshTokenGenerator
	clock                 Clock
	accessTokenTTL        time.Duration
	sessionTTL            time.Duration
	log                   *slog.Logger
}

func New(opts Options) *Service {
	return &Service{
		uow:                   opts.UoW,
		hasher:                opts.Hasher,
		accessTokenIssuer:     opts.AccessTokenIssuer,
		refreshTokenGenerator: opts.RefreshGenerator,
		clock:                 opts.Clock,
		accessTokenTTL:        opts.AccessTokenTTL,
		sessionTTL:            opts.SessionTTL,
		log:                   opts.Log,
	}
}

type RegisterInput struct {
	Email    string
	Password string
}

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type RegisterOutput struct {
	Access  Token
	Refresh Token
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	now := s.clock.Now()

	passwordHash, err := s.hasher.Hash(ctx, in.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	u := user.New(in.Email, passwordHash, now)
	sess := session.NewSession(u.ID, s.sessionTTL, now)

	accessTokenExpiresAt := now.Add(s.accessTokenTTL)

	accessToken, err := s.accessTokenIssuer.Issue(u.ID, sess.ID, now, accessTokenExpiresAt)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshTokenValue, refreshTokenHash, err := s.refreshTokenGenerator.Generate()
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("issue refresh token: %w", err)
	}

	rt := session.NewRefreshToken(sess.ID, refreshTokenHash, now)

	if err := s.uow.Do(
		ctx, func(r Repositories) error {
			if err := r.Users.Create(ctx, u); err != nil {
				return err
			}

			if err := r.Sessions.Create(ctx, sess); err != nil {
				return err
			}

			return r.RefreshTokens.Create(ctx, rt)
		},
	); err != nil {
		return RegisterOutput{}, err
	}

	s.log.InfoContext(
		ctx, "user registered",
		slog.String("user_id", u.ID.String()),
	)

	return RegisterOutput{
		Access: Token{
			Value:     accessToken,
			ExpiresAt: accessTokenExpiresAt,
		},
		Refresh: Token{
			Value:     refreshTokenValue,
			ExpiresAt: sess.ExpiresAt,
		},
	}, nil
}
