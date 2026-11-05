package config

import (
	"context"
	"fmt"

	pconfig "github.com/dz-market/platform/config"
	"github.com/dz-market/platform/vault"
)

const envPath = ".env"

func Load(ctx context.Context) (Config, error) {
	env, err := pconfig.LoadEnv(envPath)
	if err != nil {
		return Config{}, fmt.Errorf("load env: %w", err)
	}

	vaultCfg, err := vault.ConfigFromEnv(env)
	if err != nil {
		return Config{}, fmt.Errorf("vault config: %w", err)
	}

	vaultClient, err := vault.New(ctx, vaultCfg)
	if err != nil {
		return Config{}, fmt.Errorf("vault: %w", err)
	}

	cfg, err := pconfig.Load[Config](
		ctx,
		pconfig.WithEnvPath(envPath),
		pconfig.WithResolver("vault", vault.NewResolver(vaultClient)),
	)
	if err != nil {
		return cfg, err
	}

	if cfg.Kafka.DeliveryTimeout >= cfg.Outbox.BatchTimeout {
		return cfg, fmt.Errorf("%w: kafka.delivery_timeout must be less than outbox.batch_timeout", pconfig.ErrValidation)
	}

	return cfg, nil
}
