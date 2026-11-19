package bootstrap

import (
	"fmt"

	"github.com/dz-market/svc-auth/internal/config"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/jwt"
	"github.com/dz-market/svc-auth/internal/infrastructure/security/opaque"
)

type tokens struct {
	access  access
	refresh *opaque.Generator
}

type access struct {
	key      jwt.KeyPair
	issuer   *jwt.Issuer
	verifier *jwt.Verifier
}

func buildTokens(cfg config.Auth) (tokens, error) {
	accessKey, err := jwt.ParseKeyPair(cfg.Access.PrivateKeyPEM)
	if err != nil {
		return tokens{}, fmt.Errorf("load access token key pair: %w", err)
	}

	accessIssuer, err := jwt.NewIssuer(
		jwt.IssuerOptions{
			Key:      accessKey,
			Issuer:   cfg.Access.Issuer,
			Audience: cfg.Access.Audience,
		},
	)
	if err != nil {
		return tokens{}, fmt.Errorf("access token issuer: %w", err)
	}

	accessVerifier, err := jwt.NewVerifier(
		jwt.VerifierOptions{
			Key:      accessKey,
			Issuer:   cfg.Access.Issuer,
			Audience: cfg.Access.Audience,
		},
	)
	if err != nil {
		return tokens{}, fmt.Errorf("access token verifier: %w", err)
	}

	refreshGenerator, err := opaque.New(
		opaque.Options{
			Length: cfg.Refresh.Length,
		},
	)
	if err != nil {
		return tokens{}, fmt.Errorf("refresh token generator: %w", err)
	}

	return tokens{
		access: access{
			key:      accessKey,
			issuer:   accessIssuer,
			verifier: accessVerifier,
		},
		refresh: refreshGenerator,
	}, nil
}
