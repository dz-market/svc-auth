package health

import (
	"log/slog"
	"time"

	phealth "github.com/dz-market/platform/health"
)

type Options struct {
	Period  time.Duration
	Timeout time.Duration
}

func New(opts Options, log *slog.Logger) *phealth.Checker {
	return phealth.New(
		phealth.Options{
			Period:  opts.Period,
			Timeout: opts.Timeout,
		}, log,
	)
}
