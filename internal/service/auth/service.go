package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	"github.com/dz-market/svc-auth/internal/domain"
)

type TokenPair struct {
	Access  string
	Refresh string
}

type Service struct {
	userRepo    UserRepository
	refreshRepo RefreshTokenRepository
	hasher      PasswordHasher
	access      AccessIssuer
	refresh     RefreshGenerator
	refreshTTL  time.Duration
}

func New(
	userRepo UserRepository, refreshRepo RefreshTokenRepository,
	hasher PasswordHasher, access AccessIssuer,
	refresh RefreshGenerator, refreshTTL time.Duration,
) *Service {
	return &Service{
		userRepo:    userRepo,
		refreshRepo: refreshRepo,
		hasher:      hasher,
		access:      access,
		refresh:     refresh,
		refreshTTL:  refreshTTL,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (TokenPair, error) {
	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return TokenPair{}, fmt.Errorf("hash password: %w", err)
	}

	user := domain.NewUser(email, passwordHash)

	if err := s.userRepo.Create(ctx, user); err != nil {
		return TokenPair{}, err
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *Service) Login(ctx context.Context, email, password string) (TokenPair, error) {
	user, err := s.userRepo.FindByEmail(ctx, domain.NewEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return TokenPair{}, domain.ErrInvalidCredentials
		}

		return TokenPair{}, fmt.Errorf("find user by email: %w", err)
	}

	if err := s.hasher.Compare(user.Password, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return TokenPair{}, domain.ErrInvalidCredentials
		}

		return TokenPair{}, fmt.Errorf("compare password: %w", err)
	}

	return s.issueTokens(ctx, user.ID)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	oldHash := s.refresh.Hash(refreshToken)

	old, err := s.refreshRepo.FindByHash(ctx, oldHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return TokenPair{}, domain.ErrRefreshTokenNotFound
		}

		return TokenPair{}, fmt.Errorf("find refresh token by hash: %w", err)
	}

	if old.IsExpired(time.Now()) {
		return TokenPair{}, domain.ErrRefreshTokenExpired
	}

	newRaw, newHash, err := s.refresh.Generate()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	newToken := domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    old.UserID,
		SessionID: old.SessionID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}

	if err := s.refreshRepo.Rotate(ctx, oldHash, newToken); err != nil {
		if errors.Is(err, domain.ErrRefreshTokenReused) {
			if rErr := s.refreshRepo.RevokeSession(ctx, old.SessionID); rErr != nil {
				return TokenPair{}, fmt.Errorf("revoke compromised session: %w", rErr)
			}

			return TokenPair{}, domain.ErrRefreshTokenReused
		}

		return TokenPair{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	accessToken, err := s.access.Issue(old.UserID, old.SessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	return TokenPair{
		Access:  accessToken,
		Refresh: newRaw,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	hash := s.refresh.Hash(refreshToken)

	if err := s.refreshRepo.Revoke(ctx, hash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (s *Service) issueTokens(ctx context.Context, userID uuid.UUID) (TokenPair, error) {
	sessionID := uuid.New()

	accessToken, err := s.access.Issue(userID, sessionID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("issue access token: %w", err)
	}

	refreshRaw, refreshHash, err := s.refresh.Generate()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		SessionID: sessionID,
		TokenHash: refreshHash,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}

	if err := s.refreshRepo.Store(ctx, refreshToken); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenPair{
		Access:  accessToken,
		Refresh: refreshRaw,
	}, nil
}
