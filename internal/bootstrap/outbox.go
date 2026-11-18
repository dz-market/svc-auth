package bootstrap

import (
	"fmt"

	ppostgres "github.com/dz-market/platform/database/postgres"

	"github.com/dz-market/svc-auth/internal/infrastructure/messaging/kafka/producer"
	"github.com/dz-market/svc-auth/internal/infrastructure/persistence/postgres"
	"github.com/dz-market/svc-auth/internal/worker/outbox"
)

func (a app) newRelay() (*outbox.Relay, error) {
	uow, err := ppostgres.NewUnitOfWork(
		a.postgres, ppostgres.UnitOfWorkOptions{}, func(q ppostgres.Querier) outbox.Store {
			return postgres.NewOutboxRepository(q)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create outbox uow: %w", err)
	}

	return outbox.New(
		outbox.Options{
			UoW:          uow,
			Publisher:    producer.New(a.kafkaProducerClient),
			PollInterval: a.cfg.Outbox.PollInterval,
			BatchSize:    a.cfg.Outbox.BatchSize,
			BatchTimeout: a.cfg.Outbox.BatchTimeout,
			MaxAttempts:  a.cfg.Outbox.MaxAttempts,
			Log:          a.log,
		},
	), nil
}
