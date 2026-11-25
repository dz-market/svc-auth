package config

import (
	"log/slog"
	"time"

	pconfig "github.com/dz-market/platform/config"
)

type Config struct {
	ServiceName string `validate:"required" yaml:"service_name"`

	GRPC     GRPC     `yaml:"grpc"`
	Postgres Postgres `yaml:"postgres"`
	Kafka    Kafka    `yaml:"kafka"`
	Outbox   Outbox   `yaml:"outbox"`
	Auth     Auth     `yaml:"auth"`
	Health   Health   `yaml:"health"`
	Log      Log      `yaml:"log"`
}

type GRPC struct {
	Addr            string           `validate:"required"                          yaml:"addr"`
	Reflection      bool             `yaml:"reflection"`
	MaxRecvMsgSize  pconfig.ByteSize `validate:"required,minsize=1KB,maxsize=64MB" yaml:"max_recv_msg_size"`
	ShutdownTimeout time.Duration    `validate:"required,gt=0"                     yaml:"shutdown_timeout"`

	Keepalive Keepalive `yaml:"keepalive"`
}

type Keepalive struct {
	MaxConnectionAge      time.Duration `validate:"required,gt=0" yaml:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `validate:"required,gt=0" yaml:"max_connection_age_grace"`
}

type Postgres struct {
	DSN               string        `validate:"required"                yaml:"dsn"`
	MaxConns          int32         `validate:"gt=0"                    yaml:"max_conns"`
	MinConns          int32         `validate:"gte=0,ltefield=MaxConns" yaml:"min_conns"`
	MaxConnLifetime   time.Duration `validate:"gt=0"                    yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `validate:"gt=0"                    yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `validate:"gt=0"                    yaml:"health_check_period"`
	ConnectTimeout    time.Duration `validate:"gt=0"                    yaml:"connect_timeout"`
	PingTimeout       time.Duration `validate:"gt=0"                    yaml:"ping_timeout"`
}

type Kafka struct {
	Brokers         []string      `validate:"required,min=1,dive,required" yaml:"brokers"`
	DeliveryTimeout time.Duration `validate:"required,gt=0"                yaml:"delivery_timeout"`
}

type Outbox struct {
	PollInterval      time.Duration     `validate:"required,min=100ms" yaml:"poll_interval"`
	BatchSize         int               `validate:"required,min=1"     yaml:"batch_size"`
	BatchTimeout      time.Duration     `validate:"required,min=1s"    yaml:"batch_timeout"`
	MaxAttempts       int               `validate:"required,min=1"     yaml:"max_attempts"`
	PublisherRecovery PublisherRecovery `yaml:"publisher_recovery"`
}

type PublisherRecovery struct {
	CheckTimeout   time.Duration `validate:"required,gt=0"                         yaml:"check_timeout"`
	InitialBackoff time.Duration `validate:"required,gt=0"                         yaml:"initial_backoff"`
	MaxBackoff     time.Duration `validate:"required,gt=0,gtefield=InitialBackoff" yaml:"max_backoff"`
}

type Auth struct {
	Access   Access   `yaml:"access"`
	Refresh  Refresh  `yaml:"refresh"`
	Session  Session  `yaml:"session"`
	Password Password `yaml:"password"`
}

type Access struct {
	TTL           time.Duration `validate:"required,gt=0"                yaml:"ttl"`
	Issuer        string        `validate:"required"                     yaml:"issuer"`
	Audience      []string      `validate:"required,min=1,dive,required" yaml:"audience"`
	PrivateKeyPEM string        `validate:"required"                     yaml:"private_key"`
}

type Refresh struct {
	Length int `validate:"required,min=32" yaml:"length"`
}

type Session struct {
	TTL time.Duration `validate:"required,gt=0" yaml:"ttl"`
}

type Password struct {
	Argon2ID Argon2ID `yaml:"argon2id"`
}

type Argon2ID struct {
	Memory      pconfig.ByteSize `validate:"required,minsize=15MB,maxsize=1GB" yaml:"memory"`
	Iterations  uint32           `validate:"required,min=1"                    yaml:"iterations"`
	Parallelism uint8            `validate:"required,min=1"                    yaml:"parallelism"`
	SaltLength  uint32           `validate:"required,min=16"                   yaml:"salt_length"`
	KeyLength   uint32           `validate:"required,min=32"                   yaml:"key_length"`
	MaxInFlight int              `validate:"required,min=1"                    yaml:"max_in_flight"`
}

type Health struct {
	Period  time.Duration `validate:"required,gt=0" yaml:"period"`
	Timeout time.Duration `validate:"required,gt=0" yaml:"timeout"`
}

type Log struct {
	Level  slog.Level `yaml:"level"`
	Format string     `validate:"required,oneof=json text" yaml:"format"`
}
