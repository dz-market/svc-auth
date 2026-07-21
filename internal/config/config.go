package config

import (
	"time"
)

type Config struct {
	HTTP            HTTP          `yaml:"http"`
	GRPC            GRPC          `yaml:"grpc"`
	Postgres        Postgres      `yaml:"postgres"`
	Tokens          Tokens        `yaml:"tokens"`
	Log             Log           `yaml:"log"`
	ShutdownTimeout time.Duration `validate:"gt=0" yaml:"shutdown_timeout"`
}

type HTTP struct {
	Addr              string        `validate:"required" yaml:"addr"`
	ReadHeaderTimeout time.Duration `validate:"gt=0"     yaml:"read_header_timeout"`
	ReadTimeout       time.Duration `validate:"gt=0"     yaml:"read_timeout"`
	WriteTimeout      time.Duration `validate:"gt=0"     yaml:"write_timeout"`
	IdleTimeout       time.Duration `validate:"gt=0"     yaml:"idle_timeout"`
}

type GRPC struct {
	Addr string `validate:"required" yaml:"addr"`
}

type Postgres struct {
	DSN             string        `validate:"required"                yaml:"dsn"`
	MinConns        int32         `validate:"gte=0,ltefield=MaxConns" yaml:"min_conns"`
	MaxConns        int32         `validate:"gt=0"                    yaml:"max_conns"`
	ConnectTimeout  time.Duration `validate:"gt=0"                    yaml:"connect_timeout"`
	MaxConnLifetime time.Duration `validate:"gt=0"                    yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `validate:"gt=0"                    yaml:"max_conn_idle_time"`
}

type Tokens struct {
	Access  AccessToken  `yaml:"access"`
	Refresh RefreshToken `yaml:"refresh"`
}

type AccessToken struct {
	Secret string        `validate:"required,min=32" yaml:"secret"`
	TTL    time.Duration `validate:"gt=0"            yaml:"ttl"`
}

type RefreshToken struct {
	TTL     time.Duration `validate:"gt=0"   yaml:"ttl"`
	ByteLen int           `validate:"gte=32" yaml:"byte_len"`
}

type Log struct {
	Level  string `validate:"required,sloglevel"       yaml:"level"`
	Format string `validate:"required,oneof=json text" yaml:"format"`
}
