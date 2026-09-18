package interceptor

import (
	"context"
	"log/slog"
	"runtime/debug"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Recovery(log *slog.Logger) grpc.UnaryServerInterceptor {
	return recovery.UnaryServerInterceptor(
		recovery.WithRecoveryHandlerContext(
			func(ctx context.Context, p any) (err error) {
				log.ErrorContext(
					ctx, "panic recovered",
					slog.Any("panic", p),
					slog.String("stack", string(debug.Stack())),
				)

				return status.Error(codes.Internal, "internal error")
			},
		),
	)
}
