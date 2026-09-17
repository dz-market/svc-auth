package config

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

func decode[T any](tree map[string]any) (cfg T, err error) {
	dec, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           &cfg,
			TagName:          "yaml",
			WeaklyTypedInput: true,
			ErrorUnused:      true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
		},
	)
	if err != nil {
		return cfg, fmt.Errorf("build decoder: %w", err)
	}

	if err := dec.Decode(tree); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}
