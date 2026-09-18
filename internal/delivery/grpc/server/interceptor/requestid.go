package interceptor

import (
	"context"
	"log/slog"
	"uuid"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
)

const HeaderRequestID = "x-request-id"

type requestIDKey struct{}

func RequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		id := incomingRequestID(ctx)

		ctx = context.WithValue(ctx, requestIDKey{}, id)
		ctx = logger.With(ctx, slog.String("request_id", id))

		_ = grpc.SetHeader(ctx, metadata.Pairs(HeaderRequestID, id))

		return handler(ctx, req)
	}
}

func incomingRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(HeaderRequestID); len(values) > 0 && values[0] != "" {
			return values[0]
		}
	}

	return uuid.NewV4().String()
}
