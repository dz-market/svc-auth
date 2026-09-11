package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/dz-market/svc-auth/internal/config"
	logger "github.com/dz-market/svc-auth/internal/infrastructure/observability/logger/slog"
)

const serviceName = "svc-auth"

func Run(ctx context.Context, version string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(
		logger.Options{
			Level:   cfg.Log.Level,
			Format:  logger.Format(cfg.Log.Format),
			Service: serviceName,
			Version: version,
		},
	)

	log.InfoContext(
		ctx, "starting",
		slog.String("addr", cfg.GRPC.Addr),
		slog.Bool("reflection", cfg.GRPC.Reflection),
	)

	<-ctx.Done()

	log.InfoContext(ctx, "shutting down", slog.Duration("timeout", cfg.ShutdownTimeout))

	return nil
}
