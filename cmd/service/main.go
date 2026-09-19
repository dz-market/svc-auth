package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dz-market/svc-auth/internal/bootstrap"
)

var version = "dev"

func main() {
	if err := bootstrap.Run(context.Background(), version); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
