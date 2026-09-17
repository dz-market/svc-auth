package config

import (
	"log/slog"
	"reflect"
	"testing"
	"time"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	tree := map[string]any{
		"duration": "10s",
		"nested": map[string]any{
			"string": "value",
			"bool":   "true",
			"level":  "debug",
			"list":   []any{"a", "b"},
		},
	}

	got, err := decode[testConfig](tree)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	want := testConfig{
		Duration: 10 * time.Second,
		Nested: testNested{
			String: "value",
			Bool:   true,
			Level:  slog.LevelDebug,
			List:   []string{"a", "b"},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("decode() = %+v, want = %+v", got, want)
	}
}

func TestDecodeRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tree map[string]any
	}{
		{name: "unknown top-level key", tree: map[string]any{"unknown": "value"}},
		{name: "unknown nested key", tree: map[string]any{"nested": map[string]any{"unknown": "value"}}},
		{name: "unparsable duration", tree: map[string]any{"duration": "second"}},
		{name: "unparsable log level", tree: map[string]any{"nested": map[string]any{"level": "unknown"}}},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				if _, err := decode[testConfig](tt.tree); err == nil {
					t.Error("decode: want error, got nil")
				}
			},
		)
	}
}

func TestDecodeMissingLevelDefaultsToInfo(t *testing.T) {
	t.Parallel()

	got, err := decode[testConfig](map[string]any{"nested": map[string]any{"string": "value"}})
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.Nested.Level != slog.LevelInfo {
		t.Errorf("Nested.Level = %v, want INFO", got.Nested.Level)
	}
}
