package bootstrap

import (
	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka/producer"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

func (a app) newRelay(tx outbox.Transactor, store outbox.Store) *outbox.Relay {
	return outbox.New(
		outbox.Options{
			Tx:           tx,
			Store:        store,
			Publisher:    producer.New(a.kafkaProducerClient),
			PollInterval: a.cfg.Outbox.PollInterval,
			BatchSize:    a.cfg.Outbox.BatchSize,
			BatchTimeout: a.cfg.Outbox.BatchTimeout,
			MaxAttempts:  a.cfg.Outbox.MaxAttempts,
			Log:          a.log,
		},
	)
}
