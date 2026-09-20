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
	Login(ctx context.Context, in auth.LoginInput) (auth.LoginOutput, error)
	Refresh(ctx context.Context, in auth.RefreshInput) (auth.RefreshOutput, error)
}
