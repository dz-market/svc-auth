package config

import (
	"log/slog"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		ShutdownTimeout: 10 * time.Second,
		GRPC:            GRPC{Addr: ":50051"},
		Log:             Log{Level: slog.LevelInfo, Format: "json"},
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{name: "complete", mutate: func(*Config) {}},
		{name: "no shutdown timeout", mutate: func(c *Config) { c.ShutdownTimeout = 0 }, wantErr: true},
		{name: "no grpc address", mutate: func(c *Config) { c.GRPC.Addr = "" }, wantErr: true},
		{name: "no log format", mutate: func(c *Config) { c.Log.Format = "" }, wantErr: true},
		{name: "unsupported log format", mutate: func(c *Config) { c.Log.Format = "unsupported" }, wantErr: true},
		{name: "text log format", mutate: func(c *Config) { c.Log.Format = "text" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tt.mutate(&cfg)

			err := cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
