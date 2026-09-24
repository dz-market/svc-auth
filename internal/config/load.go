package config

import pconfig "github.com/dz-market/platform/config"

func Load() (Config, error) {
	return pconfig.Load[Config]()
}
