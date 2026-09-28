package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"uuid"
)

const (
	minRetryDelay = time.Second
	maxRetryDelay = 5 * time.Minute
)

type Options struct {
	UoW          UnitOfWork
	Publisher    Publisher
	PollInterval time.Duration
	BatchSize    int
	BatchTimeout time.Duration
	MaxAttempts  int
	Log          *slog.Logger
}

type Relay struct {
	uow          UnitOfWork
	publisher    Publisher
	pollInterval time.Duration
	batchSize    int
	batchTimeout time.Duration
	maxAttempts  int
	log          *slog.Logger
}

func New(opts Options) *Relay {
	return &Relay{
		uow:          opts.UoW,
		publisher:    opts.Publisher,
		pollInterval: opts.PollInterval,
		batchSize:    opts.BatchSize,
		batchTimeout: opts.BatchTimeout,
		maxAttempts:  opts.MaxAttempts,
		log:          opts.Log,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	for {
		r.drain(ctx)

		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
		}
	}
}

func (r *Relay) drain(ctx context.Context) {
	for ctx.Err() == nil {
		more, err := r.publishBatch(ctx)
		if err != nil {
			r.log.ErrorContext(
				ctx, "outbox relay failed",
				slog.Any("err", err),
			)

			return
		}

		if !more {
			return
		}
	}
}

func (r *Relay) publishBatch(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.batchTimeout)
	defer cancel()

	var more bool

	err := r.uow.Do(
		ctx, func(s Store) error {
			now := time.Now().UTC()

			msgs, err := s.Unpublished(ctx, r.batchSize, now)
			if err != nil {
				return fmt.Errorf("load unpublished messages: %w", err)
			}

			if len(msgs) == 0 {
				return nil
			}

			sent, failed := r.publish(ctx, msgs, now)

			if len(sent) > 0 {
				if err := s.MarkPublished(ctx, sent, now); err != nil {
					return fmt.Errorf("mark published messages: %w", err)
				}
			}

			if len(failed) > 0 {
				if err := s.MarkFailed(ctx, failed, now); err != nil {
					return fmt.Errorf("mark failed messages: %w", err)
				}
			}

			more = len(msgs) == r.batchSize

			return nil
		},
	)

	return more, err
}

func (r *Relay) publish(ctx context.Context, msgs []Message, now time.Time) ([]uuid.UUID, []Failure) {
	attempts := make(map[uuid.UUID]int, len(msgs))
	for _, msg := range msgs {
		attempts[msg.ID] = msg.Attempts + 1
	}

	var (
		sent   = make([]uuid.UUID, 0, len(msgs))
		failed []Failure
	)

	for _, res := range r.publisher.Publish(ctx, msgs) {
		if res.Err == nil {
			sent = append(sent, res.ID)

			continue
		}

		f := Failure{
			ID:  res.ID,
			Err: res.Err.Error(),
		}
		if attempts[res.ID] < r.maxAttempts {
			f.RetryAt = now.Add(retryDelay(attempts[res.ID]))
		}

		r.log.WarnContext(
			ctx, "outbox message not published",
			slog.String("id", res.ID.String()),
			slog.Int("attempt", attempts[res.ID]),
			slog.Bool("gave_up", f.RetryAt.IsZero()),
			slog.Any("err", res.Err),
		)

		failed = append(failed, f)
	}

	return sent, failed
}

func retryDelay(attempt int) time.Duration {
	return min(minRetryDelay<<min(attempt-1, 20), maxRetryDelay)
}
