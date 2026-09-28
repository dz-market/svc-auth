package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

const TopicUserRegistered = "auth.user-registered.v1"

const ensureTopicsRetryDelay = 5 * time.Second

type Topic struct {
	Name       string
	Partitions int32
}

func EnsureTopics(ctx context.Context, cl *kgo.Client, topics []Topic, log *slog.Logger) {
	adm := kadm.NewClient(cl)

	for {
		err := createTopics(ctx, adm, topics)
		if err == nil || ctx.Err() != nil {
			return
		}

		log.WarnContext(
			ctx, "create kafka topics failed, retrying",
			slog.Any("err", err),
			slog.Duration("delay", ensureTopicsRetryDelay),
		)

		select {
		case <-ctx.Done():
			return

		case <-time.After(ensureTopicsRetryDelay):
		}
	}
}

func createTopics(ctx context.Context, adm *kadm.Client, topics []Topic) error {
	for _, topic := range topics {
		_, err := adm.CreateTopic(ctx, topic.Partitions, -1, nil, topic.Name)
		if err != nil && !errors.Is(err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("create topic %s: %w", topic.Name, err)
		}
	}

	return nil
}
