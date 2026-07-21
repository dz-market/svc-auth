package config

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/go-viper/mapstructure/v2"
)

const (
	DefaultConfigPath = "config/config.yml"
	DefaultEnvPath    = ".env"
)

type LookupFunc func(name string) (string, bool)

type Option func(*loader)

type loader struct {
	configPath string
	envPath    string
	lookup     LookupFunc
}

func WithPath(path string) Option {
	return func(l *loader) {
		l.configPath = path
	}
}

func WithEnvFile(path string) Option {
	return func(l *loader) {
		l.envPath = path
	}
}

func WithLookup(fn LookupFunc) Option {
	return func(l *loader) {
		l.lookup = fn
	}
}

func Load(opts ...Option) (Config, error) {
	l := loader{
		configPath: DefaultConfigPath,
		envPath:    DefaultEnvPath,
	}

	for _, opt := range opts {
		opt(&l)
	}

	if l.lookup == nil {
		lookup, err := envLookup(l.envPath)
		if err != nil {
			return Config{}, err
		}

		l.lookup = lookup
	}

	raw, err := os.ReadFile(l.configPath)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var tree map[string]any
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	var missing []string

	expandLeaves(tree, l.lookup, &missing)

	if len(missing) > 0 {
		slices.Sort(missing)
		missing = slices.Compact(missing)

		return Config{}, fmt.Errorf("config: unset variables: %s", strings.Join(missing, ", "))
	}

	var cfg Config

	dec, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           &cfg,
			TagName:          "yaml",
			WeaklyTypedInput: true,
			ErrorUnused:      true,
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		},
	)
	if err != nil {
		return Config{}, fmt.Errorf("build decoder: %w", err)
	}

	if err := dec.Decode(tree); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}
