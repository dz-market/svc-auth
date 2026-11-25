package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"uuid"
)

const (
	minRetryDelay = time.Second
	maxRetryDelay = 5 * time.Minute

	failedBatchesBeforeCheck = 2
)

type relayMode uint8

const (
	relayNormal relayMode = iota
	relayPaused
	relayProbing
)

type Options struct {
	Tx                Transactor
	Repo              Repository
	Publisher         Publisher
	CheckAvailability func(context.Context) error
	Log               *slog.Logger

	PollInterval time.Duration
	BatchSize    int
	BatchTimeout time.Duration
	MaxAttempts  int

	AvailabilityCheckTimeout time.Duration
	InitialRecoveryBackoff   time.Duration
	MaxRecoveryBackoff       time.Duration
}

type Relay struct {
	tx                Transactor
	repo              Repository
	publisher         Publisher
	checkAvailability func(context.Context) error
	log               *slog.Logger

	pollInterval time.Duration
	batchSize    int
	batchTimeout time.Duration
	maxAttempts  int

	availabilityCheckTimeout time.Duration
	initialRecoveryBackoff   time.Duration
	maxRecoveryBackoff       time.Duration
}

func New(opts Options) *Relay {
	return &Relay{
		tx:                opts.Tx,
		repo:              opts.Repo,
		publisher:         opts.Publisher,
		checkAvailability: opts.CheckAvailability,
		log:               opts.Log,

		pollInterval: opts.PollInterval,
		batchSize:    opts.BatchSize,
		batchTimeout: opts.BatchTimeout,
		maxAttempts:  opts.MaxAttempts,

		availabilityCheckTimeout: opts.AvailabilityCheckTimeout,
		initialRecoveryBackoff:   opts.InitialRecoveryBackoff,
		maxRecoveryBackoff:       opts.MaxRecoveryBackoff,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	if r.checkAvailability == nil {
		return errors.New("outbox publisher availability check is required")
	}

	state := relayState{
		mode:    relayNormal,
		backoff: r.initialRecoveryBackoff,
	}

	for ctx.Err() == nil {
		if state.mode == relayPaused {
			if !waitFor(ctx, state.backoff) {
				return nil
			}

			if err := r.checkPublisher(ctx); err != nil {
				state.backoff = r.nextRecoveryBackoff(state.backoff)

				continue
			}

			state.mode = relayProbing
		}

		limit := r.batchSize

		if state.mode == relayProbing {
			limit = 1
		}

		result, err := r.publishBatch(ctx, limit)
		if err != nil {
			r.log.ErrorContext(ctx, "outbox relay failed", slog.Any("err", err))
		}

		if ctx.Err() != nil {
			return nil //nolint:nilerr // The parent context signals the relay to stop.
		}

		r.handleBatchResult(ctx, &state, result)

		if state.mode == relayPaused {
			continue
		}

		if err == nil && result.attempted == limit {
			continue
		}

		if !waitFor(ctx, r.pollInterval) {
			return nil
		}
	}

	return nil
}

type relayState struct {
	mode          relayMode
	failedBatches int
	backoff       time.Duration
}

func (r *Relay) handleBatchResult(ctx context.Context, state *relayState, result batchResult) {
	switch {
	case result.acknowledged > 0:
		if state.mode == relayProbing {
			r.log.InfoContext(ctx, "outbox publishing resumed")
		}

		state.mode = relayNormal
		state.failedBatches = 0
		state.backoff = r.initialRecoveryBackoff

	case result.attempted == 0:
		return

	case state.mode == relayProbing:
		state.backoff = r.nextRecoveryBackoff(state.backoff)
		state.mode = relayPaused

	default:
		state.failedBatches++
		if state.failedBatches < failedBatchesBeforeCheck {
			return
		}

		state.failedBatches = 0

		if err := r.checkPublisher(ctx); err != nil {
			r.log.WarnContext(ctx, "outbox publishing paused", slog.Any("err", err))

			state.mode = relayPaused
		}
	}
}

func (r *Relay) checkPublisher(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.availabilityCheckTimeout)
	defer cancel()

	return r.checkAvailability(ctx)
}

func (r *Relay) nextRecoveryBackoff(backoff time.Duration) time.Duration {
	return backoff + min(backoff, r.maxRecoveryBackoff-backoff)
}

type batchResult struct {
	attempted    int
	acknowledged int
}

func (r *Relay) publishBatch(ctx context.Context, limit int) (batchResult, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.batchTimeout)
	defer cancel()

	var result batchResult

	err := r.tx.Do(
		ctx, func(ctx context.Context) error {
			now := time.Now().UTC()

			msgs, err := r.repo.Unpublished(ctx, limit, now)
			if err != nil {
				return fmt.Errorf("load unpublished messages: %w", err)
			}

			if len(msgs) == 0 {
				return nil
			}

			sent, failed := r.publish(ctx, msgs)

			result = batchResult{
				attempted:    len(msgs),
				acknowledged: len(sent),
			}

			if len(sent) > 0 {
				if err := r.repo.MarkPublished(ctx, sent, now); err != nil {
					return fmt.Errorf("mark published messages: %w", err)
				}
			}

			if len(failed) > 0 {
				if err := r.repo.MarkFailed(ctx, failed, now); err != nil {
					return fmt.Errorf("mark failed messages: %w", err)
				}
			}

			return nil
		},
	)

	return result, err
}

func (r *Relay) publish(ctx context.Context, msgs []Message) ([]uuid.UUID, []Failure) {
	attempts := make(map[uuid.UUID]int, len(msgs))
	for _, msg := range msgs {
		attempts[msg.ID] = msg.Attempts + 1
	}

	var (
		sent   = make([]uuid.UUID, 0, len(msgs))
		failed []Failure
	)

	results := r.publisher.Publish(ctx, msgs)
	finishedAt := time.Now().UTC()

	for _, res := range results {
		if res.Err == nil {
			sent = append(sent, res.ID)

			continue
		}

		f := Failure{
			ID:  res.ID,
			Err: res.Err.Error(),
		}
		if attempts[res.ID] < r.maxAttempts {
			f.RetryAt = finishedAt.Add(retryDelay(attempts[res.ID]))
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

func waitFor(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}
