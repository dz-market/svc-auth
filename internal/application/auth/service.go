package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"

	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type Options struct {
	Repos             Repositories
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
	repos                 Repositories
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
		repos:                 opts.Repos,
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

type Token struct {
	Value     string
	ExpiresAt time.Time
}

type RegisterInput struct {
	Email    string
	Password string
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

	issued, err := s.issueSession(u.ID, now)
	if err != nil {
		return RegisterOutput{}, err
	}

	if err := s.uow.Do(
		ctx, func(r Repositories) error {
			if err := r.Users.Create(ctx, u); err != nil {
				return err
			}

			if err := r.Sessions.Create(ctx, issued.session); err != nil {
				return err
			}

			return r.RefreshTokens.Create(ctx, issued.refreshToken)
		},
	); err != nil {
		return RegisterOutput{}, err
	}

	s.log.InfoContext(
		ctx, "user registered",
		slog.String("user_id", u.ID.String()),
	)

	return RegisterOutput{
		Access:  issued.access,
		Refresh: issued.refresh,
	}, nil
}

type LoginInput struct {
	Email    string
	Password string
}

type LoginOutput struct {
	Access  Token
	Refresh Token
}

func (s *Service) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	now := s.clock.Now()

	u, err := s.repos.Users.ByEmail(ctx, user.NormalizeEmail(in.Email))
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return LoginOutput{}, user.ErrInvalidCredentials
		}

		return LoginOutput{}, fmt.Errorf("find user: %w", err)
	}

	match, err := s.hasher.Verify(ctx, in.Password, u.PasswordHash)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("verify password: %w", err)
	}

	if !match {
		return LoginOutput{}, user.ErrInvalidCredentials
	}

	issued, err := s.issueSession(u.ID, now)
	if err != nil {
		return LoginOutput{}, err
	}

	if err := s.uow.Do(
		ctx, func(r Repositories) error {
			if err := r.Sessions.Create(ctx, issued.session); err != nil {
				return err
			}

			return r.RefreshTokens.Create(ctx, issued.refreshToken)
		},
	); err != nil {
		return LoginOutput{}, err
	}

	s.log.InfoContext(
		ctx, "user logged in",
		slog.String("user_id", u.ID.String()),
	)

	return LoginOutput{
		Access:  issued.access,
		Refresh: issued.refresh,
	}, nil
}

type issuedSession struct {
	session      session.Session
	refreshToken session.RefreshToken
	access       Token
	refresh      Token
}

func (s *Service) issueSession(userID uuid.UUID, now time.Time) (issuedSession, error) {
	sess := session.NewSession(userID, s.sessionTTL, now)

	accessTokenExpiresAt := now.Add(s.accessTokenTTL)

	accessTokenValue, err := s.accessTokenIssuer.Issue(userID, sess.ID, now, accessTokenExpiresAt)
	if err != nil {
		return issuedSession{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshTokenValue, refreshTokenHash, err := s.refreshTokenGenerator.Generate()
	if err != nil {
		return issuedSession{}, fmt.Errorf("generate refresh token: %w", err)
	}

	return issuedSession{
		session:      sess,
		refreshToken: session.NewRefreshToken(sess.ID, refreshTokenHash, now),
		access: Token{
			Value:     accessTokenValue,
			ExpiresAt: accessTokenExpiresAt,
		},
		refresh: Token{
			Value:     refreshTokenValue,
			ExpiresAt: sess.ExpiresAt,
		},
	}, nil
}
