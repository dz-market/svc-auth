package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

func envLookup(envPath string) (LookupFunc, error) {
	vars, err := godotenv.Read(envPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", envPath, err)
	}

	return func(name string) (string, bool) {
		if v, ok := os.LookupEnv(name); ok {
			return v, true
		}

		v, ok := vars[name]

		return v, ok
	}, nil
}
