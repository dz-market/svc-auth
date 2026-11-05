package bootstrap

import (
	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka/producer"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

func (a app) newRelay() *outbox.Relay {
	return outbox.New(
		outbox.Options{
			UoW: ppostgres.NewUnitOfWork(
				a.postgres, func(q ppostgres.Querier) outbox.Store {
					return postgres.NewOutboxRepository(q)
				},
			),
			Publisher:    producer.New(a.kafkaProducerClient),
			PollInterval: a.cfg.Outbox.PollInterval,
			BatchSize:    a.cfg.Outbox.BatchSize,
			BatchTimeout: a.cfg.Outbox.BatchTimeout,
			MaxAttempts:  a.cfg.Outbox.MaxAttempts,
			Log:          a.log,
		},
	)
}
