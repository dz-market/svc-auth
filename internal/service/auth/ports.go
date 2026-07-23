package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dz-market/svc-auth/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
	FindByEmail(ctx context.Context, email domain.Email) (domain.User, error)
}

type RefreshTokenRepository interface {
	Store(ctx context.Context, t domain.RefreshToken) error
	FindByHash(ctx context.Context, hash []byte) (domain.RefreshToken, error)
	Rotate(ctx context.Context, oldHash []byte, newToken domain.RefreshToken) error
	Revoke(ctx context.Context, hash []byte) error
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type AccessIssuer interface {
	Issue(userID, sessionID uuid.UUID) (string, time.Time, error)
}

type RefreshGenerator interface {
	Generate() (raw string, hash []byte, err error)
	Hash(raw string) []byte
}
