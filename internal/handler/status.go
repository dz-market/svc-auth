package handler

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dz-market/svc-auth/internal/domain"
)

func toStatus(err error) error {
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		return status.Error(codes.AlreadyExists, "email already taken")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, "invalid credentials")
	case errors.Is(err, domain.ErrRefreshTokenExpired):
		return status.Error(codes.Unauthenticated, "refresh token expired")
	case errors.Is(err, domain.ErrRefreshTokenNotFound), errors.Is(err, domain.ErrRefreshTokenReused):
		return status.Error(codes.Unauthenticated, "invalid refresh token")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
