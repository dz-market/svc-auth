package slog

import (
	"log/slog"

	plogger "github.com/dz-market/platform/logger/slog"
)

type Options struct {
	Level   slog.Level
	Format  string
	Service string
	Version string
}

func New(opts Options) *slog.Logger {
	return plogger.New(
		plogger.Options{
			Level:   opts.Level,
			Format:  plogger.Format(opts.Format),
			Service: opts.Service,
			Version: opts.Version,
		},
	)
}
