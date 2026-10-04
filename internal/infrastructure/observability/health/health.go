package health

import (
	"log/slog"
	"time"

	phealth "github.com/dz-market/platform/health"
)

type Checker = phealth.Checker

type Options struct {
	Period  time.Duration
	Timeout time.Duration
}

func New(opts Options, log *slog.Logger) *Checker {
	return phealth.New(
		phealth.Options{
			Period:  opts.Period,
			Timeout: opts.Timeout,
		}, log,
	)
}
