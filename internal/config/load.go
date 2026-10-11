package config

import (
	"context"
	"fmt"

	pconfig "github.com/dz-market/platform/config"
)

func Load(ctx context.Context) (Config, error) {
	cfg, err := pconfig.Load[Config](ctx)
	if err != nil {
		return cfg, err
	}

	if cfg.Kafka.DeliveryTimeout >= cfg.Outbox.BatchTimeout {
		return cfg, fmt.Errorf("%w: kafka.delivery_timeout must be less than outbox.batch_timeout", pconfig.ErrValidation)
	}

	return cfg, nil
}
