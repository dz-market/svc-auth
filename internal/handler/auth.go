package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/dz-market/proto/gen/go/auth/v1"

	"github.com/dz-market/svc-auth/internal/service/auth"
)

type AuthHandler struct {
	authv1.UnimplementedAuthServiceServer

	svc *auth.Service
}

func NewAuthHandler(svc *auth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	userID, tokens, err := h.svc.Register(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatus(err)
	}

	return authv1.RegisterResponse_builder{
		UserId: new(userID.String()),
		Tokens: authv1.TokenPair_builder{
			AccessToken:      new(tokens.Access),
			AccessExpiresAt:  timestamppb.New(tokens.AccessExpiresAt),
			RefreshToken:     new(tokens.Refresh),
			RefreshExpiresAt: timestamppb.New(tokens.RefreshExpiresAt),
		}.Build(),
	}.Build(), nil
}

func (h *AuthHandler) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	tokens, err := h.svc.Login(ctx, req.GetEmail(), req.GetPassword())
	if err != nil {
		return nil, toStatus(err)
	}

	return authv1.LoginResponse_builder{
		Tokens: authv1.TokenPair_builder{
			AccessToken:      new(tokens.Access),
			AccessExpiresAt:  timestamppb.New(tokens.AccessExpiresAt),
			RefreshToken:     new(tokens.Refresh),
			RefreshExpiresAt: timestamppb.New(tokens.RefreshExpiresAt),
		}.Build(),
	}.Build(), nil
}

func (h *AuthHandler) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	tokens, err := h.svc.Refresh(ctx, req.GetRefreshToken())
	if err != nil {
		return nil, toStatus(err)
	}

	return authv1.RefreshResponse_builder{
		Tokens: authv1.TokenPair_builder{
			AccessToken:      new(tokens.Access),
			AccessExpiresAt:  timestamppb.New(tokens.AccessExpiresAt),
			RefreshToken:     new(tokens.Refresh),
			RefreshExpiresAt: timestamppb.New(tokens.RefreshExpiresAt),
		}.Build(),
	}.Build(), nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if err := h.svc.Logout(ctx, req.GetRefreshToken()); err != nil {
		return nil, toStatus(err)
	}

	return &authv1.LogoutResponse{}, nil
}
