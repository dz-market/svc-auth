package interceptor

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dz-market/svc-auth/internal/delivery/grpc/identity"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
)

const schemeBearer = "bearer"

var errInvalidAccessToken = status.Error(codes.Unauthenticated, "invalid access token")

type TokenVerifier interface {
	Verify(token string) (userID, sessionID uuid.UUID, err error)
}

func Auth(verifier TokenVerifier) grpc.UnaryServerInterceptor {
	return auth.UnaryServerInterceptor(
		func(ctx context.Context) (context.Context, error) {
			token, err := auth.AuthFromMD(ctx, schemeBearer)
			if err != nil {
				return nil, errInvalidAccessToken
			}

			userID, sessionID, err := verifier.Verify(token)
			if err != nil {
				return nil, errInvalidAccessToken
			}

			ctx = identity.With(
				ctx, identity.Identity{
					SessionID: sessionID,
				},
			)

			ctx = logger.With(ctx, slog.String("user_id", userID.String()))

			return ctx, nil
		},
	)
}
