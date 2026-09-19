package config

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

func (c Config) validate() error {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(
		func(f reflect.StructField) string {
			return f.Tag.Get("yaml")
		},
	)

	return v.Struct(c)
}
