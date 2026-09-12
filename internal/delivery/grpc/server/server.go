package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Options struct {
	Addr       string
	Reflection bool
}

type Server struct {
	grpc   *grpc.Server
	health *health.Server
	addr   string
	log    *slog.Logger
}

func New(opts Options, log *slog.Logger) *Server {
	srv := grpc.NewServer()

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)

	if opts.Reflection {
		reflection.Register(srv)
		log.Warn("grpc reflection is enabled")
	}

	return &Server{
		grpc:   srv,
		health: healthSrv,
		addr:   opts.Addr,
		log:    log,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	var lc net.ListenConfig

	lis, err := lc.Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}

	s.health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	s.log.InfoContext(ctx, "grpc server listening", slog.String("addr", s.addr))

	if err := s.grpc.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context, timeout time.Duration) {
	s.health.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	done := make(chan struct{})

	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.log.InfoContext(ctx, "grpc server stopped gracefully")

	case <-ctx.Done():
		s.log.WarnContext(ctx, "grpc server did not drain in time, forcing stop", slog.Duration("timeout", timeout))
		s.grpc.Stop()
	}
}

func (s *Server) SetServing(healthy bool) {
	status := healthpb.HealthCheckResponse_NOT_SERVING
	if healthy {
		status = healthpb.HealthCheckResponse_SERVING
	}

	s.health.SetServingStatus("", status)
}
