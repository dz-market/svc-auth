package health

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"
)

type Probe func(ctx context.Context) error

type check struct {
	name  string
	probe Probe
	up    bool
}

type Checker struct {
	timeout time.Duration
	log     *slog.Logger

	mu     sync.Mutex
	checks []*check

	healthy bool
}

func New(log *slog.Logger, timeout time.Duration) *Checker {
	return &Checker{
		timeout: timeout,
		log:     log,
		healthy: true,
	}
}

func (c *Checker) Register(name string, probe Probe) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.checks = append(c.checks, &check{name: name, probe: probe, up: true})
}

func (c *Checker) Run(ctx context.Context, period time.Duration, onChange func(healthy bool)) error {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	c.evaluate(ctx, onChange)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			c.evaluate(ctx, onChange)
		}
	}
}

func (c *Checker) evaluate(ctx context.Context, onChange func(bool)) {
	c.mu.Lock()
	checks := slices.Clone(c.checks)
	c.mu.Unlock()

	healthy := true

	for _, check := range checks {
		err := c.probe(ctx, check.probe)
		up := err == nil

		if !up {
			healthy = false
		}

		if up == check.up {
			c.log.DebugContext(
				ctx, "health check completed",
				slog.String("dependency", check.name),
			)

			continue
		}

		check.up = up

		if up {
			c.log.InfoContext(
				ctx, "dependency restored",
				slog.String("dependency", check.name),
			)

			continue
		}

		c.log.ErrorContext(
			ctx, "dependency unavailable",
			slog.String("dependency", check.name),
			slog.Any("err", err),
		)
	}

	if healthy == c.healthy {
		return
	}

	c.healthy = healthy
	onChange(healthy)
}

func (c *Checker) probe(ctx context.Context, probe Probe) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	return probe(ctx)
}
