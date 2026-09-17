package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/mapper"
	"github.com/dz-market/svc-auth/internal/domain/user"
)

type AuthService interface {
	Register(ctx context.Context, in auth.RegisterInput) (auth.RegisterOutput, error)
}

type Auth struct {
	authv1.UnimplementedAuthServiceServer

	service AuthService
	log     *slog.Logger
}

func NewAuth(service AuthService, log *slog.Logger) *Auth {
	return &Auth{
		service: service,
		log:     log,
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
