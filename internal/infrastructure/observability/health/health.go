package health

import (
	"context"
	"log/slog"
	"time"
)

type Checker interface {
	Ping(ctx context.Context) error
}

type Notifier func(healthy bool)

type Options struct {
	Period  time.Duration
	Timeout time.Duration
}

type Monitor struct {
	checks  map[string]Checker
	states  map[string]bool
	notify  []Notifier
	period  time.Duration
	timeout time.Duration
	healthy bool
	log     *slog.Logger
}

func New(opts Options, log *slog.Logger) *Monitor {
	return &Monitor{
		checks:  make(map[string]Checker),
		states:  make(map[string]bool),
		period:  opts.Period,
		timeout: opts.Timeout,
		healthy: true,
		log:     log,
	}
}

func (m *Monitor) Register(name string, check Checker) {
	m.checks[name] = check
	m.states[name] = true
}

func (m *Monitor) OnChange(fn Notifier) {
	m.notify = append(m.notify, fn)
}

func (m *Monitor) Run(ctx context.Context) {
	ticker := time.NewTicker(m.period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			m.check(ctx)
		}
	}
}

func (m *Monitor) check(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	healthy := true

	for name, check := range m.checks {
		err := check.Ping(ctx)
		up := err == nil

		if !up {
			healthy = false
		}

		if up == m.states[name] {
			continue
		}

		m.states[name] = up

		if up {
			m.log.InfoContext(ctx, "dependency restored", "dependency", name)

			continue
		}

		m.log.ErrorContext(
			ctx, "dependency unavailable",
			"dependency", name,
			"err", err,
		)
	}

	if healthy == m.healthy {
		return
	}

	m.healthy = healthy

	for _, fn := range m.notify {
		fn(healthy)
	}
}
