package interceptor

import (
	"context"
	"log/slog"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func Logging(log *slog.Logger) grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(
		slogAdapter(log),
		logging.WithLogOnEvents(logging.FinishCall),
		logging.WithLevels(codeToLevel),
		logging.WithDisableLoggingFields("protocol", "grpc.component", "grpc.method_type"),
	)
}

func slogAdapter(log *slog.Logger) logging.Logger {
	return logging.LoggerFunc(
		func(ctx context.Context, level logging.Level, msg string, fields ...any) {
			log.Log(ctx, slog.Level(level), msg, fields...)
		},
	)
}

func codeToLevel(code codes.Code) logging.Level {
	switch code {
	case codes.OK, codes.NotFound, codes.Canceled, codes.AlreadyExists,
		codes.InvalidArgument, codes.Unauthenticated:
		return logging.LevelInfo

	case codes.DeadlineExceeded, codes.PermissionDenied, codes.ResourceExhausted,
		codes.FailedPrecondition, codes.Aborted, codes.OutOfRange, codes.Unavailable:
		return logging.LevelWarn

	default:
		return logging.LevelError
	}
}
