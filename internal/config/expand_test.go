package config

import (
	"reflect"
	"slices"
	"testing"
)

func testLookup(name string) (string, bool) {
	v, ok := map[string]string{
		"SET":   "value",
		"EMPTY": "",
	}[name]

	return v, ok
}

func TestExpand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          string
		want        string
		wantMissing []string
	}{
		{name: "no references", in: "plain text", want: "plain text"},
		{name: "resolved", in: "${SET}", want: "value"},
		{name: "resolved to empty value", in: "${EMPTY}", want: ""},
		{name: "inside text", in: "host=${SET}", want: "host=value"},
		{name: "repeated", in: "${SET}/${SET}", want: "value/value"},
		{name: "default applied", in: "${UNSET:-fallback}", want: "fallback"},
		{name: "default ignored when set", in: "${SET:-fallback}", want: "value"},
		{name: "empty default", in: "${UNSET:-}", want: ""},
		{name: "escaped reference", in: "$${SET}", want: "${SET}"},
		{name: "unresolved", in: "${UNSET}", wantMissing: []string{"UNSET"}},
		{name: "empty name", in: "${}", wantMissing: []string{"${}"}},
		{name: "several unresolved", in: "${A}${B}", wantMissing: []string{"A", "B"}},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				var missing []string

				got := expand(tt.in, testLookup, &missing)
				if got != tt.want {
					t.Errorf("expand(%q) = %q, want %q", tt.in, got, tt.want)
				}

				if !slices.Equal(missing, tt.wantMissing) {
					t.Errorf("expand(%q) missing = %v, want %v", tt.in, missing, tt.wantMissing)
				}
			},
		)
	}
}

func TestExpandNode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		node        any
		want        any
		wantMissing []string
	}{
		{name: "string is expanded", node: "${SET}", want: "value"},
		{name: "number is left alone", node: 10, want: 10},
		{name: "bool is left alone", node: true, want: true},
		{
			name: "map is expanded recursively",
			node: map[string]any{"key": "${SET}"},
			want: map[string]any{"key": "value"},
		},
		{
			name: "list is expanded element by element",
			node: []any{"${SET}", "plain"},
			want: []any{"value", "plain"},
		},
		{
			name:        "unresolved deep inside is collected",
			node:        map[string]any{"outer": map[string]any{"inner": "${UNSET}"}},
			want:        map[string]any{"outer": map[string]any{"inner": ""}},
			wantMissing: []string{"UNSET"},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				t.Parallel()

				var missing []string

				got := expandNode(tt.node, testLookup, &missing)

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("expandNode() = %#v, want %#v", got, tt.want)
				}

				if !slices.Equal(missing, tt.wantMissing) {
					t.Errorf("missing = %v, want %v", missing, tt.wantMissing)
				}
			},
		)
	}
}
