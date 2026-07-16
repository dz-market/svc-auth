package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/dz-market/svc-auth/internal/config"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		return fmt.Errorf("parse log level: %w", err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()
	mux.HandleFunc(
		"GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
	mux.HandleFunc(
		"GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           mux,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		log.Info("http listening", "addr", cfg.HTTP.Addr)

		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}

	shCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}
