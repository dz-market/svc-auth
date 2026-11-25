package outbox_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dz-market/svc-auth/internal/worker/outbox"
	"github.com/dz-market/svc-auth/internal/worker/outbox/mocks"
)

const (
	pollInterval    = time.Second
	batchSize       = 2
	batchTimeout    = 10 * time.Second
	maxAttempts     = 3
	recoveryBackoff = 5 * time.Second
)

//nolint:gochecknoglobals // test fixtures
var (
	fixedNow = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	errFailed = errors.New("failed")
)

type testRelay struct {
	relay     *outbox.Relay
	tx        *mocks.MockTransactor
	publisher *mocks.MockPublisher
	repo      *mocks.MockRepository
}

func newTestRelay(t *testing.T) testRelay {
	t.Helper()

	return newTestRelayWithAvailability(
		t, func(context.Context) error {
			return nil
		},
	)
}

func newTestRelayWithAvailability(t *testing.T, checkAvailability func(context.Context) error) testRelay {
	t.Helper()

	r := testRelay{
		tx:        mocks.NewMockTransactor(t),
		publisher: mocks.NewMockPublisher(t),
		repo:      mocks.NewMockRepository(t),
	}

	r.tx.EXPECT().
		Do(mock.Anything, mock.Anything).
		RunAndReturn(
			func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			},
		).
		Maybe()

	r.relay = outbox.New(
		outbox.Options{
			Tx:                r.tx,
			Repo:              r.repo,
			Publisher:         r.publisher,
			CheckAvailability: checkAvailability,
			Log:               slog.New(slog.DiscardHandler),

			PollInterval: pollInterval,
			BatchSize:    batchSize,
			BatchTimeout: batchTimeout,
			MaxAttempts:  maxAttempts,

			AvailabilityCheckTimeout: 2 * time.Second,
			InitialRecoveryBackoff:   recoveryBackoff,
			MaxRecoveryBackoff:       time.Minute,
		},
	)

	return r
}

func (r testRelay) start(t *testing.T) (stop func()) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		done <- r.relay.Run(ctx)
	}()

	return func() {
		cancel()
		require.NoError(t, <-done)
	}
}

func newMessage(attempts int) outbox.Message {
	return outbox.Message{
		ID:       uuid.NewV7(),
		Topic:    "topic",
		Key:      "key",
		Payload:  []byte("payload"),
		Attempts: attempts,
	}
}

func results(msgs []outbox.Message, errs ...error) []outbox.Result {
	res := make([]outbox.Result, 0, len(msgs))

	for i, msg := range msgs {
		r := outbox.Result{ID: msg.ID}

		if i < len(errs) {
			r.Err = errs[i]
		}

		res = append(res, r)
	}

	return res
}

func TestRelay_PublishesDueMessages(t *testing.T) {
	t.Parallel()

	synctest.Test(
		t, func(t *testing.T) {
			r := newTestRelay(t)
			msgs := []outbox.Message{newMessage(0)}

			r.repo.EXPECT().
				Unpublished(mock.Anything, batchSize, fixedNow).
				Return(msgs, nil).
				Once()

			r.publisher.EXPECT().
				Publish(mock.Anything, msgs).
				Return(results(msgs)).
				Once()

			r.repo.EXPECT().
				MarkPublished(mock.Anything, []uuid.UUID{msgs[0].ID}, fixedNow).
				Return(nil).
				Once()

			stop := r.start(t)

			synctest.Wait()
			stop()
		},
	)
}

func TestRelay_NothingDue(t *testing.T) {
	t.Parallel()

	synctest.Test(
		t, func(t *testing.T) {
			r := newTestRelay(t)

			r.repo.EXPECT().
				Unpublished(mock.Anything, batchSize, fixedNow).
				Return(nil, nil).
				Once()

			stop := r.start(t)

			synctest.Wait()
			stop()
		},
	)
}

func TestRelay_Failures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveAttempts int
		wantRetryAt  time.Time
	}{
		{
			name:         "first failure is retried after a second",
			giveAttempts: 0,
			wantRetryAt:  fixedNow.Add(time.Second),
		},
		{
			name:         "retry delay doubles with every attempt",
			giveAttempts: 1,
			wantRetryAt:  fixedNow.Add(2 * time.Second),
		},
		{
			name:         "last attempt gives up",
			giveAttempts: maxAttempts - 1,
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				synctest.Test(
					t, func(t *testing.T) {
						r := newTestRelay(t)
						msgs := []outbox.Message{newMessage(tt.giveAttempts), newMessage(0)}

						r.repo.EXPECT().
							Unpublished(mock.Anything, batchSize, fixedNow).
							Return(msgs, nil).
							Once()

						r.repo.EXPECT().
							Unpublished(mock.Anything, batchSize, fixedNow).
							Return(nil, nil).
							Once()

						r.publisher.EXPECT().
							Publish(mock.Anything, msgs).
							Return(results(msgs, errFailed)).
							Once()

						r.repo.EXPECT().
							MarkPublished(mock.Anything, []uuid.UUID{msgs[1].ID}, fixedNow).
							Return(nil).
							Once()

						r.repo.EXPECT().
							MarkFailed(
								mock.Anything, []outbox.Failure{
									{
										ID:      msgs[0].ID,
										Err:     errFailed.Error(),
										RetryAt: tt.wantRetryAt,
									},
								}, fixedNow,
							).
							Return(nil).
							Once()

						stop := r.start(t)

						synctest.Wait()
						stop()
					},
				)
			},
		)
	}
}

func TestRelay_DrainsFullBatches(t *testing.T) {
	t.Parallel()

	synctest.Test(
		t, func(t *testing.T) {
			r := newTestRelay(t)
			full := []outbox.Message{newMessage(0), newMessage(0)}
			partial := []outbox.Message{newMessage(0)}

			for _, msgs := range [][]outbox.Message{full, partial} {
				r.repo.EXPECT().
					Unpublished(mock.Anything, batchSize, fixedNow).
					Return(msgs, nil).
					Once()

				r.publisher.EXPECT().
					Publish(mock.Anything, msgs).
					Return(results(msgs)).
					Once()

				r.repo.EXPECT().
					MarkPublished(mock.Anything, mock.Anything, fixedNow).
					Return(nil).
					Once()
			}

			stop := r.start(t)

			synctest.Wait()
			stop()
		},
	)
}

func TestRelay_PollsAfterStoreError(t *testing.T) {
	t.Parallel()

	synctest.Test(
		t, func(t *testing.T) {
			r := newTestRelay(t)

			r.repo.EXPECT().
				Unpublished(mock.Anything, batchSize, fixedNow).
				Return(nil, errFailed).
				Once()

			r.repo.EXPECT().
				Unpublished(mock.Anything, batchSize, fixedNow.Add(pollInterval)).
				Return(nil, nil).
				Once()

			stop := r.start(t)

			synctest.Wait()

			time.Sleep(pollInterval)
			synctest.Wait()
			stop()
		},
	)
}

func TestRelay_FinishesBatchOnShutdown(t *testing.T) {
	t.Parallel()

	synctest.Test(
		t, func(t *testing.T) {
			r := newTestRelay(t)
			ctx, cancel := context.WithCancel(t.Context())
			msgs := []outbox.Message{newMessage(0), newMessage(0)}

			r.repo.EXPECT().
				Unpublished(mock.Anything, batchSize, fixedNow).
				Return(msgs, nil).
				Once()

			r.publisher.EXPECT().
				Publish(mock.Anything, msgs).
				RunAndReturn(
					func(context.Context, []outbox.Message) []outbox.Result {
						cancel()

						return results(msgs)
					},
				).
				Once()

			r.repo.EXPECT().
				MarkPublished(mock.Anything, mock.Anything, fixedNow).
				RunAndReturn(
					func(ctx context.Context, _ []uuid.UUID, _ time.Time) error {
						assert.NoError(t, ctx.Err())

						return nil
					},
				).
				Once()

			require.NoError(t, r.relay.Run(ctx))
		},
	)
}
