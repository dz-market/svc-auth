package slog

import (
	"log/slog"

	"github.com/dz-market/platform/logger"
)

type Options struct {
	Format  string
	Level   slog.Level
	Service string
	Version string
}

func New(opts Options) (*slog.Logger, error) {
	return logger.New(
		logger.Options{
			Format: logger.Format(opts.Format),
			HandlerOptions: slog.HandlerOptions{
				Level: opts.Level,
			},
			Service: opts.Service,
			Version: opts.Version,
		},
	)
}
