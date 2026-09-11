package config

import (
	"log/slog"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tree := map[string]any{
		"shutdown_timeout": "10s",
		"grpc": map[string]any{
			"addr":       ":50051",
			"reflection": "true",
		},
		"log": map[string]any{
			"level":  "debug",
			"format": "json",
		},
	}

	cfg, err := decode(tree)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", cfg.ShutdownTimeout)
	}

	if cfg.GRPC.Addr != ":50051" {
		t.Errorf("GRPC.Addr = %v, want %q", cfg.GRPC.Addr, ":50051")
	}

	if !cfg.GRPC.Reflection {
		t.Error("GRPC.Reflection = false, want true")
	}

	if cfg.Log.Level != slog.LevelDebug {
		t.Errorf("Log.Level = %v, want DEBUG", cfg.Log.Level)
	}

	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %v, want %q", cfg.Log.Format, "json")
	}
}

func TestDecodeRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tree map[string]any
	}{
		{name: "unknown top-level key", tree: map[string]any{"unknown": "value"}},
		{name: "unknown nested key", tree: map[string]any{"grpc": map[string]any{"unknown": "value"}}},
		{name: "unparsable duration", tree: map[string]any{"shutdown_timeout": "second"}},
		{name: "unparsable log level", tree: map[string]any{"log": map[string]any{"level": "unknown"}}},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				if _, err := decode(tt.tree); err == nil {
					t.Error("decode: want error, got nil")
				}
			},
		)
	}
}

func TestDecodeMissingLevelDefaultsToInfo(t *testing.T) {
	t.Parallel()

	cfg, err := decode(map[string]any{"log": map[string]any{"format": "json"}})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if cfg.Log.Level != slog.LevelInfo {
		t.Errorf("Log.Level = %v, want INFO", cfg.Log.Level)
	}
}
