package postgres

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	ppostgres "github.com/dz-market/platform/database/postgres"
	autheventsv1 "github.com/dz-market/protobuf/gen/go/auth/events/v1"

	"github.com/dz-market/svc-auth/internal/domain/user"
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

type OutboxRepository struct {
	q ppostgres.Querier
}

func NewOutboxRepository(q ppostgres.Querier) *OutboxRepository {
	return &OutboxRepository{
		q: q,
	}
}

func (r *OutboxRepository) AddUserRegistered(ctx context.Context, e user.Registered) error {
	payload, err := proto.Marshal(
		autheventsv1.UserRegistered_builder{
			EventId:      new(e.EventID.String()),
			UserId:       new(e.UserID.String()),
			RegisteredAt: timestamppb.New(e.RegisteredAt),
		}.Build(),
	)
	if err != nil {
		return fmt.Errorf("marshal user registered event: %w", err)
	}

	const query = `
		INSERT INTO outbox (id, topic, key, payload)
		VALUES ($1, $2, $3, $4)
	`

	if _, err := r.q.Exec(ctx, query, e.EventID, kafka.TopicUserRegistered, e.UserID.String(), payload); err != nil {
		return fmt.Errorf("add user registered to outbox: %w", err)
	}

	return nil
}

func (r *OutboxRepository) Unpublished(ctx context.Context, limit int, now time.Time) ([]outbox.Message, error) {
	const query = `
		SELECT id, topic, key, payload, attempts
		FROM outbox
		WHERE published_at IS NULL
			AND failed_at IS NULL
			AND next_attempt_at <= $2
		ORDER BY created_at, id
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.q.Query(ctx, query, limit, now)
	if err != nil {
		return nil, fmt.Errorf("query unpublished outbox: %w", err)
	}

	msgs, err := pgx.CollectRows(
		rows, func(row pgx.CollectableRow) (outbox.Message, error) {
			var m outbox.Message

			err := row.Scan(&m.ID, &m.Topic, &m.Key, &m.Payload, &m.Attempts)

			return m, err
		},
	)
	if err != nil {
		return nil, fmt.Errorf("scan unpublished outbox: %w", err)
	}

	return msgs, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, ids []uuid.UUID, at time.Time) error {
	const query = `
		UPDATE outbox
		SET published_at = $2
		WHERE id = ANY($1)
	`

	if _, err := r.q.Exec(ctx, query, ids, at); err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}

	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, failures []outbox.Failure, at time.Time) error {
	const query = `
		UPDATE outbox
		SET attempts = attempts + 1,
		    last_error = $2,
		    next_attempt_at = coalesce($3, next_attempt_at),
		    failed_at = $4
		WHERE id = $1
	`

	for _, f := range failures {
		var retryAt, failedAt *time.Time

		if f.RetryAt.IsZero() {
			failedAt = &at
		} else {
			retryAt = &f.RetryAt
		}

		if _, err := r.q.Exec(ctx, query, f.ID, f.Err, retryAt, failedAt); err != nil {
			return fmt.Errorf("mark outbox failed: %w", err)
		}
	}

	return nil
}
