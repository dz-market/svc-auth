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
	DSN               string        `validate:"required" yaml:"dsn"`
	MaxConns          int32         `yaml:"max_conns"`
	MinConns          int32         `yaml:"min_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`
	PingTimeout       time.Duration `yaml:"ping_timeout"`
}

type Log struct {
	Level  slog.Level `yaml:"level"`
	Format string     `validate:"required,oneof=json text" yaml:"format"`
}
