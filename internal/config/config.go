package config

import (
	"log/slog"
	"time"
)

type Config struct {
	ShutdownTimeout time.Duration `validate:"required" yaml:"shutdown_timeout"`

	GRPC     GRPC     `yaml:"grpc"`
	Postgres Postgres `yaml:"postgres"`
	Log      Log      `yaml:"log"`
}

type GRPC struct {
	Addr       string `validate:"required" yaml:"addr"`
	Reflection bool   `yaml:"reflection"`
}

type Postgres struct {
	DSN string `yaml:"dsn"`
}

type Log struct {
	Level  slog.Level `yaml:"level"`
	Format string     `validate:"required,oneof=json text" yaml:"format"`
}
