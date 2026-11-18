package server

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"buf.build/go/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	validation "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/protovalidate"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/selector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pgrpc "github.com/dz-market/platform/grpc"
	pinterceptor "github.com/dz-market/platform/grpc/interceptor"
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
	addr            string
	shutdownTimeout time.Duration
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
		grpc.ChainUnaryInterceptor(
			pinterceptor.RequestID(),
			pinterceptor.Logging(log),
			pinterceptor.Recovery(log),
			auth,
			validation.UnaryServerInterceptor(opts.Validator),
		),
		grpc.KeepaliveParams(
			keepalive.ServerParameters{
				MaxConnectionAge:      opts.MaxConnectionAge,
				MaxConnectionAgeGrace: opts.MaxConnectionAgeGrace,
			},
		),
		grpc.MaxRecvMsgSize(opts.MaxRecvMsgSize),
	)

	if opts.Reflection {
		reflection.Register(srv)

		log.Warn("grpc reflection is enabled")
	}

	return &Server{
		grpc:            srv,
		addr:            opts.Addr,
		shutdownTimeout: opts.ShutdownTimeout,
	}
}

func (s *Server) Run(ctx context.Context, beforeStop func()) error {
	return pgrpc.ListenAndServe(
		ctx, s.grpc, s.addr, pgrpc.ServeOptions{
			ShutdownTimeout: s.shutdownTimeout,
			BeforeStop:      beforeStop,
		},
	)
}

func (s *Server) Registrar() grpc.ServiceRegistrar {
	return s.grpc
}
