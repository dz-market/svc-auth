package outbox

import (
	"context"
	"time"
	"uuid"
)

type Message struct {
	ID       uuid.UUID
	Topic    string
	Key      string
	Payload  []byte
	Attempts int
}

type Failure struct {
	ID      uuid.UUID
	Err     string
	RetryAt time.Time
}

type Result struct {
	ID  uuid.UUID
	Err error
}

type Store interface {
	Unpublished(ctx context.Context, limit int, now time.Time) ([]Message, error)
	MarkPublished(ctx context.Context, ids []uuid.UUID, at time.Time) error
	MarkFailed(ctx context.Context, failures []Failure, at time.Time) error
}

type UnitOfWork interface {
	Do(ctx context.Context, fn func(s Store) error) error
}

type Publisher interface {
	Publish(ctx context.Context, msgs []Message) []Result
}

type Clock interface {
	Now() time.Time
}
