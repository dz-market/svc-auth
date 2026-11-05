package bootstrap

import (
	"fmt"

	"buf.build/go/protovalidate"
	jwtgo "github.com/golang-jwt/jwt/v5"

	authv1 "github.com/dz-market/protobuf/gen/go/auth/api/v1"

	"github.com/dz-market/svc-auth/internal/application/auth"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/handler"
	"github.com/dz-market/svc-auth/internal/delivery/grpc/server"
)

func (a app) newGRPCServer(tokens tokenProviders, authService *auth.Service) (*server.Server, error) {
	validator, err := protovalidate.New()
	if err != nil {
		return nil, fmt.Errorf("protovalidate: %w", err)
	}

	srv := server.New(
		server.Options{
			Addr:                  a.cfg.GRPC.Addr,
			Reflection:            a.cfg.GRPC.Reflection,
			MaxRecvMsgSize:        a.cfg.GRPC.MaxRecvMsgSize.Bytes(),
			MaxConnectionAge:      a.cfg.GRPC.Keepalive.MaxConnectionAge,
			MaxConnectionAgeGrace: a.cfg.GRPC.Keepalive.MaxConnectionAgeGrace,
			ShutdownTimeout:       a.cfg.ShutdownTimeout,
			Validator:             validator,
			Verifier:              tokens.accessTokenVerifier,
		},
		a.log,
	)

	authv1.RegisterAuthServiceServer(
		srv.Registrar(), handler.NewAuth(
			handler.Options{
				Service: authService,
				PublicKeys: []handler.PublicKey{
					{
						ID:        tokens.accessKey.ID,
						Algorithm: jwtgo.SigningMethodRS256.Alg(),
						Key:       tokens.accessKey.Public,
					},
				},
				Log: a.log,
			},
		),
	)

	return srv, nil
}
