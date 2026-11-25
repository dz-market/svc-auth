package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"buf.build/go/protovalidate"
	jwtgo "github.com/golang-jwt/jwt/v5"

	phealth "github.com/dz-market/platform/grpc/health"
	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/handler"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/server"
)

func buildGRPC(
	cfg config.Config, accessToken access, authService *auth.Service,
	pingPostgres func(context.Context) error, log *slog.Logger,
) (*server.Server, *phealth.Checker, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, nil, fmt.Errorf("protovalidate: %w", err)
	}

	srv := server.New(
		server.Options{
			Addr:                  cfg.GRPC.Addr,
			Reflection:            cfg.GRPC.Reflection,
			MaxRecvMsgSize:        cfg.GRPC.MaxRecvMsgSize.Bytes(),
			MaxConnectionAge:      cfg.GRPC.Keepalive.MaxConnectionAge,
			MaxConnectionAgeGrace: cfg.GRPC.Keepalive.MaxConnectionAgeGrace,
			ShutdownTimeout:       cfg.GRPC.ShutdownTimeout,
			Validator:             validator,
			Verifier:              accessToken.verifier,
		},
		log,
	)

	authHandler := handler.NewAuth(
		handler.Options{
			Service: authService,
			PublicKeys: []handler.PublicKey{
				{
					ID:        accessToken.key.ID,
					Algorithm: jwtgo.SigningMethodRS256.Alg(),
					Key:       accessToken.key.Public,
				},
			},
			Log: log,
		},
	)

	authv1.RegisterAuthServiceServer(srv.Registrar(), authHandler)

	checker, err := phealth.New(
		srv.Registrar(), phealth.Options{
			Period:  cfg.Health.Period,
			Timeout: cfg.Health.Timeout,
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create grpc health checker: %w", err)
	}

	if err := checker.Register("postgres", phealth.DependencyOptions{}, pingPostgres); err != nil {
		return nil, nil, fmt.Errorf("register postgres health check: %w", err)
	}

	return srv, checker, nil
}
