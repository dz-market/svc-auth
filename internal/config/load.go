package config

import (
	"context"
	"fmt"

	pconfig "github.com/dz-market/platform/config"
	vaultresolver "github.com/dz-market/platform/config/resolver/vault"
	"github.com/dz-market/platform/vault"
)

const (
	envPath      = ".env"
	vaultAddrEnv = "VAULT_ADDR"
)

func Load(ctx context.Context) (Config, error) {
	env, err := pconfig.LoadEnv(envPath)
	if err != nil {
		return Config{}, fmt.Errorf("load env: %w", err)
	}

	vaultAddr, err := env.Required(vaultAddrEnv)
	if err != nil {
		return Config{}, err
	}

	vaultClient, err := vault.NewWithAddr(vaultAddr)
	if err != nil {
		return Config{}, fmt.Errorf("create vault client: %w", err)
	}

	cfg, err := pconfig.Load[Config](
		ctx,
		pconfig.WithEnvPath(envPath),
		pconfig.WithResolver("vault", vaultresolver.New(vaultClient)),
	)
	if err != nil {
		return cfg, err
	}

	if cfg.Kafka.DeliveryTimeout >= cfg.Outbox.BatchTimeout {
		return cfg, fmt.Errorf("%w: kafka.delivery_timeout must be less than outbox.batch_timeout", pconfig.ErrValidation)
	}

	return cfg, nil
}
