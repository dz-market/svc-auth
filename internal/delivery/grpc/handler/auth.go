package handler

import (
	"context"
	"crypto/rsa"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/identity"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/mapper"
	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type PublicKey struct {
	ID        string
	Algorithm string
	Key       *rsa.PublicKey
}

type Options struct {
	Service    AuthService
	PublicKeys []PublicKey
	Log        *slog.Logger
}

type Auth struct {
	authv1.UnimplementedAuthServiceServer

	service    AuthService
	publicKeys []PublicKey
	log        *slog.Logger
}

func NewAuth(opts Options) *Auth {
	return &Auth{
		service:    opts.Service,
		publicKeys: opts.PublicKeys,
		log:        opts.Log,
	}
}

func (h *Auth) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	out, err := h.service.Register(
		ctx, auth.RegisterInput{
			Email:    req.GetEmail(),
			Password: req.GetPassword(),
		},
	)
	if err != nil {
		return nil, h.toStatus(ctx, err)
	}

	return mapper.ToRegisterResponse(out, time.Now().UTC()), nil
}

func (h *Auth) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	out, err := h.service.Login(
		ctx, auth.LoginInput{
			Email:    req.GetEmail(),
			Password: req.GetPassword(),
		},
	)
	if err != nil {
		return nil, h.toStatus(ctx, err)
	}

	return mapper.ToLoginResponse(out, time.Now().UTC()), nil
}

func (h *Auth) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	out, err := h.service.Refresh(
		ctx, auth.RefreshInput{
			RefreshToken: req.GetRefreshToken(),
		},
	)
	if err != nil {
		return nil, h.toStatus(ctx, err)
	}

	return mapper.ToRefreshResponse(out, time.Now().UTC()), nil
}

func (h *Auth) Logout(ctx context.Context, _ *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	id, ok := identity.From(ctx)
	if !ok {
		h.log.ErrorContext(ctx, "identity is missing from the context")

		return nil, status.Errorf(codes.Unauthenticated, "invalid access token")
	}

	if err := h.service.Logout(
		ctx, auth.LogoutInput{
			SessionID: id.SessionID,
		},
	); err != nil {
		return nil, h.toStatus(ctx, err)
	}

	return authv1.LogoutResponse_builder{}.Build(), nil
}

func (h *Auth) GetJwks(_ context.Context, _ *authv1.GetJwksRequest) (*authv1.GetJwksResponse, error) {
	keys := make([]*authv1.Jwk, 0, len(h.publicKeys))

	for _, key := range h.publicKeys {
		keys = append(keys, mapper.ToJwk(key.ID, key.Algorithm, key.Key))
	}

	return authv1.GetJwksResponse_builder{
		Keys: keys,
	}.Build(), nil
}

func (h *Auth) toStatus(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, user.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, user.ErrEmailTaken.Error())

	case errors.Is(err, user.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, user.ErrInvalidCredentials.Error())

	case errors.Is(err, session.ErrInvalidRefreshToken):
		return status.Error(codes.Unauthenticated, session.ErrInvalidRefreshToken.Error())

	default:
		h.log.ErrorContext(
			ctx, "unhandled error",
			slog.Any("err", err),
		)

		return status.Error(codes.Internal, "internal error")
	}
}
