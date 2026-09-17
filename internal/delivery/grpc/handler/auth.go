package handler

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/mapper"
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

func (h *Auth) toStatus(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, user.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, "email is already registered")

	default:
		h.log.ErrorContext(
			ctx, "unhandled error",
			slog.Any("err", err),
		)

		return status.Error(codes.Internal, "internal error")
	}
}
