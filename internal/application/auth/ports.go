package auth

import (
	"context"
	"time"
	"uuid"

	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type Clock interface {
	Now() time.Time
}

type UsersRepository interface {
	Create(ctx context.Context, u user.User) error
	ByEmail(ctx context.Context, email string) (user.User, error)
}

type RefreshTokensRepository interface {
	Create(ctx context.Context, t session.RefreshToken) error
}

type SessionsRepository interface {
	Create(ctx context.Context, s session.Session) error
}

type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Verify(ctx context.Context, password, passwordHash string) (bool, error)
}

type AccessTokenIssuer interface {
	Issue(userID, sessionID uuid.UUID, issuedAt, expiresAt time.Time) (string, error)
}

type RefreshTokenGenerator interface {
	Generate() (string, []byte, error)
	Fingerprint(value string) []byte
}

type Repositories struct {
	Users         UsersRepository
	RefreshTokens RefreshTokensRepository
	Sessions      SessionsRepository
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(r Repositories) error) error
}
