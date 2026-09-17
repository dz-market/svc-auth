package config

import (
	"fmt"
	"reflect"

	"github.com/docker/go-units"
	"github.com/go-playground/validator/v10"
)

func validate(cfg any) error {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(
		func(f reflect.StructField) string {
			return f.Tag.Get("yaml")
		},
	)

	if err := v.RegisterValidation("minsize", sizeAtLeast); err != nil {
		return fmt.Errorf("register minsize: %w", err)
	}

	if err := v.RegisterValidation("maxsize", sizeAtMost); err != nil {
		return fmt.Errorf("register maxsize: %w", err)
	}

	return v.Struct(cfg)
}

func sizeAtLeast(fl validator.FieldLevel) bool {
	limit, err := units.RAMInBytes(fl.Param())

	return err == nil && fl.Field().Int() >= limit
}

func sizeAtMost(fl validator.FieldLevel) bool {
	limit, err := units.RAMInBytes(fl.Param())

	return err == nil && fl.Field().Int() <= limit
}
