package config

import (
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func noLookup(string) (string, bool) { return "", false }

func TestLoadDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := Load(
		WithPath(filepath.Join("testdata", "config.yml")),
		withLookup(noLookup),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Config{
		ShutdownTimeout: 10 * time.Second,
		GRPC:            GRPC{Addr: ":50051", Reflection: false},
		Log:             Log{Level: slog.LevelInfo, Format: "json"},
	}

	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	t.Parallel()

	lookup := func(name string) (string, bool) {
		v, ok := map[string]string{
			"SHUTDOWN_TIMEOUT": "30s",
			"GRPC_ADDR":        ":50052",
			"GRPC_REFLECTION":  "true",
			"LOG_LEVEL":        "debug",
			"LOG_FORMAT":       "text",
		}[name]

		return v, ok
	}

	cfg, err := Load(
		WithPath(filepath.Join("testdata", "config.yml")),
		withLookup(lookup),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := Config{
		ShutdownTimeout: 30 * time.Second,
		GRPC:            GRPC{Addr: ":50052", Reflection: true},
		Log:             Log{Level: slog.LevelDebug, Format: "text"},
	}

	if cfg != want {
		t.Errorf("Load() = %+v, want %+v", cfg, want)
	}
}

func TestLoadErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		file     string
		wantText string
	}{
		{name: "no such file", file: "absent.yml", wantText: "read config"},
		{name: "malformed yaml", file: "broken.yml", wantText: "parse config"},
		{name: "unknown key", file: "unknown.yml", wantText: "decode config"},
		{name: "required field missing", file: "incomplete.yml", wantText: "validate config"},
		{
			name:     "unresolved references",
			file:     "unresolved.yml",
			wantText: "unresolved references: GRPC_ADDR, LOG_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := Load(
					WithPath(filepath.Join("testdata", tt.file)),
					withLookup(noLookup),
				)
				if err == nil {
					t.Fatal("Load() want error, got nil")
				}

				if !strings.Contains(err.Error(), tt.wantText) {
					t.Errorf("Load() error = %q, want it to contain %q", err, tt.wantText)
				}
			},
		)
	}
}

func TestLoadReadsEnvFile(t *testing.T) {
	path := writeEnv(t, "GRPC_ADDR=:50052")

	cfg, err := Load(
		WithPath(filepath.Join("testdata", "config.yml")),
		WithEnvPath(path),
	)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.GRPC.Addr != ":50052" {
		t.Errorf("GRPC.Addr = %q, want = %q", cfg.GRPC.Addr, ":50052")
	}
}
