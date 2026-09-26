package server

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"buf.build/go/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pinterceptor "github.com/dz-market/platform/grpc/interceptor"
	pserver "github.com/dz-market/platform/grpc/server"
	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/delivery/grpc/server/interceptor"
)

type Options struct {
	Addr                  string
	Reflection            bool
	MaxRecvMsgSize        int
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	ShutdownTimeout       time.Duration
	Validator             protovalidate.Validator
	Verifier              interceptor.TokenVerifier
}

type Server struct {
	grpc            *grpc.Server
	health          *health.Server
	reportHealth    func(healthy bool)
	addr            string
	shutdownTimeout time.Duration
	log             *slog.Logger
}

func New(opts Options, log *slog.Logger) *Server {
	protectedMethods := []string{
		authv1.AuthService_Logout_FullMethodName,
	}

	auth := selector.UnaryServerInterceptor(
		interceptor.Auth(opts.Verifier),
		selector.MatchFunc(
			func(_ context.Context, c interceptors.CallMeta) bool {
				return slices.Contains(protectedMethods, c.FullMethod())
			},
		),
	)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(pinterceptor.Default(log, opts.Validator, auth)...),
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionAge:      opts.MaxConnectionAge,
				MaxConnectionAgeGrace: opts.MaxConnectionAgeGrace,
			},
		),
		grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize),
	)

	healthSrv := pserver.RegisterHealth(srv)

	if opts.Reflection {
		reflection.Register(srv)

		log.Warn("grpc reflection is enabled")
	}

	return &Server{
		grpc:            srv,
		health:          healthSrv,
		reportHealth:    pserver.ReportHealth(healthSrv),
		addr:            opts.Addr,
		shutdownTimeout: opts.ShutdownTimeout,
		log:             log,
	}
}

func (s *Server) Run(ctx context.Context) error {
	return pserver.ListenAndServe(
		ctx, s.grpc, s.addr, pserver.ServeOptions{
			ShutdownTimeout: s.shutdownTimeout,
			BeforeStop:      s.health.Shutdown,
		}, s.log,
	)
}

func (s *Server) SetHealthy(healthy bool) {
	s.reportHealth(healthy)
}

func (s *Server) Registrar() grpc.ServiceRegistrar {
	return s.grpc
}
