package config

import (
	"log/slog"
	"time"
)

type Config struct {
	ServiceName     string        `validate:"required" yaml:"service_name"`
	ShutdownTimeout time.Duration `validate:"required" yaml:"shutdown_timeout"`

	GRPC     GRPC     `yaml:"grpc"`
	Postgres Postgres `yaml:"postgres"`
	Auth     Auth     `yaml:"auth"`
	Health   Health   `yaml:"health"`
	Log      Log      `yaml:"log"`
}

type GRPC struct {
	Addr       string `validate:"required" yaml:"addr"`
	Reflection bool   `yaml:"reflection"`
}

type Postgres struct {
	DSN               string        `validate:"required"        yaml:"dsn"`
	MaxConns          int32         `yaml:"max_conns"`
	MinConns          int32         `yaml:"min_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`
	PingTimeout       time.Duration `yaml:"ping_timeout"`
}

type Auth struct {
	Access   Access   `yaml:"access"`
	Refresh  Refresh  `yaml:"refresh"`
	Session  Session  `yaml:"session"`
	Password Password `yaml:"password"`
}

type Access struct {
	TTL            time.Duration `validate:"required"                     yaml:"ttl"`
	Issuer         string        `validate:"required"                     yaml:"issuer"`
	Audience       []string      `validate:"required,min=1,dive,required" yaml:"audience"`
	PrivateKeyPath string        `validate:"required"                     yaml:"private_key_path"`
	PublicKeyPath  string        `validate:"required"                     yaml:"public_key_path"`
}

type Refresh struct {
	Length int `validate:"required,min=32" yaml:"length"`
}

type Session struct {
	TTL time.Duration `validate:"required" yaml:"ttl"`
}

type Password struct {
	Argon2ID Argon2ID `yaml:"argon2id"`
}

type Argon2ID struct {
	Memory      ByteSize `validate:"required,minsize=15MB,maxsize=1GB" yaml:"memory"`
	Iterations  uint32   `validate:"required,min=1"                    yaml:"iterations"`
	Parallelism uint8    `validate:"required,min=1"                    yaml:"parallelism"`
	SaltLength  uint32   `validate:"required,min=16"                   yaml:"salt_length"`
	KeyLength   uint32   `validate:"required,min=32"                   yaml:"key_length"`
	MaxInFlight int      `validate:"required,min=1"                    yaml:"max_in_flight"`
}

type Health struct {
	Period  time.Duration `validate:"required" yaml:"period"`
	Timeout time.Duration `validate:"required" yaml:"timeout"`
}

type Log struct {
	Level  slog.Level `yaml:"level"`
	Format string     `validate:"required,oneof=json text" yaml:"format"`
}
