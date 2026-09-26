package postgres

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	ppostgres "github.com/dz-market/platform/database/postgres"
	autheventsv1 "github.com/dz-market/protobuf/gen/go/auth/events/v1"

	"github.com/dz-market/svc-auth/internal/domain/user"
)

const topicUserRegistered = "auth.user-registered.v1"

type OutboxRepository struct {
	q ppostgres.Querier
}

func NewOutboxRepository(q ppostgres.Querier) *OutboxRepository {
	return &OutboxRepository{
		q: q,
	}
}

func (r *OutboxRepository) AddUserRegistered(ctx context.Context, e user.Registered) error {
	event := &autheventsv1.UserRegistered{}
	event.SetEventId(e.EventID.String())
	event.SetUserId(e.UserID.String())
	event.SetRegisteredAt(timestamppb.New(e.RegisteredAt))

	payload, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal user registered event: %w", err)
	}

	const query = `
		INSERT INTO outbox (id, topic, key, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	if _, err := r.q.Exec(ctx, query, e.EventID, topicUserRegistered, e.UserID.String(), payload, e.RegisteredAt); err != nil {
		return fmt.Errorf("add user registered to outbox: %w", err)
	}

	return nil
}
