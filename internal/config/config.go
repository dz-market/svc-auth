package config

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	HTTP            HTTP          `koanf:"http"`
	Log             Log           `koanf:"log"`
	ShutdownTimeout time.Duration `koanf:"shutdown_timeout"`
}

type HTTP struct {
	Addr              string        `koanf:"addr"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	IdleTimeout       time.Duration `koanf:"idle_timeout"`
}

type Log struct {
	Level string `koanf:"level"`
}

func Load() (Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider("config/config.yml"), yaml.Parser()); err != nil {
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}
