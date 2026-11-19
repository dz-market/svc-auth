package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/dz-market/svc-auth/internal/bootstrap"
)

var buildVersion string

func main() {
	if err := bootstrap.Run(context.Background(), resolveVersion()); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}

func resolveVersion() string {
	if buildVersion != "" {
		return buildVersion
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if version := info.Main.Version; version != "" && version != "(devel)" {
			return version
		}
	}

	return "dev"
}
