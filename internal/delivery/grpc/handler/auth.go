package handler

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/identity"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/mapper"
	"github.com/dz-market/svc-auth/internal/domain/session"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type Options struct {
	Service AuthService
	Clock   Clock
	Log     *slog.Logger
}

type Auth struct {
	authv1.UnimplementedAuthServiceServer

	service AuthService
	clock   Clock
	log     *slog.Logger
}

func NewAuth(opts Options) *Auth {
	return &Auth{
		service: opts.Service,
		clock:   opts.Clock,
		log:     opts.Log,
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

	return mapper.ToRegisterResponse(out, h.clock.Now()), nil
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

	return mapper.ToLoginResponse(out, h.clock.Now()), nil
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

	return mapper.ToRefreshResponse(out, h.clock.Now()), nil
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
