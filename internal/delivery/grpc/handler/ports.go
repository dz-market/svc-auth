package handler

import (
	"context"
	"time"

	"github.com/dz-market/svc-auth/internal/application/auth"
)

type Clock interface {
	Now() time.Time
}

type AuthService interface {
	Register(ctx context.Context, in auth.RegisterInput) (auth.RegisterOutput, error)
}
