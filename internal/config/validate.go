package config

import (
	"fmt"
	"log/slog"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())

	if err := v.RegisterValidation(
		"sloglevel", func(fl validator.FieldLevel) bool {
			var lvl slog.Level

			return lvl.UnmarshalText([]byte(fl.Field().String())) == nil
		},
	); err != nil {
		panic(fmt.Sprintf("config: register sloglevel validator: %v", err))
	}

	return v
}

func (c Config) Validate() error {
	return validate.Struct(c)
}
